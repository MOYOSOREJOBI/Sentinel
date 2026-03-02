package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sentinel/internal/audit"
	"sentinel/internal/auth"
	"sentinel/internal/authz"
	"sentinel/internal/cache"
	"sentinel/internal/config"
	"sentinel/internal/db"
	"sentinel/internal/healthcheck"
	"sentinel/internal/httpx"
	"sentinel/internal/logging"
	"sentinel/internal/metrics"
	"sentinel/internal/middleware"
	"sentinel/internal/replay"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8084)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.New("governance")
	cfg := config.Load("governance")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("governance: invalid config: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("governance: failed to connect to database: %v", err)
	}
	defer pool.Close()

	pub, err := auth.ReadPublic(cfg.JWTPublicKey)
	if err != nil {
		log.Fatalf("governance: failed to read JWT public key: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)
	r.Use(metrics.HTTPMiddleware("governance"))
	r.Use(middleware.RequireCSRFFunc(authz.SubjectFromRequest(pub)))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"service": "governance",
			"version": "dev",
			"links": map[string]string{
				"healthz": "/healthz",
				"readyz":  "/readyz",
				"metrics": "/metrics",
				"docs":    "/docs",
			},
		})
	})
	r.Get("/readyz", readinessHandler(pool))
	read := r.With(authz.RequirePerm(pub, "read"))
	admin := r.With(middleware.RateLimit(rateLimitSubjectKey("gov", pub), 10, 1*time.Minute), authz.RequireAnyPerm(pub, "*", "model:deploy"))
	workflow := r.With(middleware.RateLimit(rateLimitSubjectKey("workflow", pub), 30, 1*time.Minute), authz.RequireAnyPerm(pub, "replay:write", "*"))
	policyAdmin := r.With(middleware.RateLimit(rateLimitSubjectKey("policy", pub), 10, 1*time.Minute), authz.RequireAnyPerm(pub, "*"))

	read.Get("/governance/summary", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"modelLineage":       loadModelRegistryRows(r.Context(), pool),
			"modelDeployments":   loadModelDeploymentRows(r.Context(), pool),
			"replayJobs":         loadReplayJobRows(r.Context(), pool),
			"escalationPolicies": loadEscalationPolicyRows(r.Context(), pool),
			"thresholdChanges":   []any{},
		})
	})

	read.Get("/active-models", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(ctx, `SELECT model_name,model_version,artifact_hash,artifact_path,feature_set_version,coalesce(calibration_version,''),deployed_at FROM model_deployments WHERE status='deployed' ORDER BY deployed_at DESC NULLS LAST`)
		if err != nil {
			logger.Error("load active models failed", map[string]any{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := map[string]any{"models": []map[string]any{}}
		models := []map[string]any{}
		for rows.Next() {
			var name, version, artifactHash, artifactPath, featureSetVersion, calibrationVersion string
			var deployedAt any
			if err := rows.Scan(&name, &version, &artifactHash, &artifactPath, &featureSetVersion, &calibrationVersion, &deployedAt); err != nil {
				continue
			}
			models = append(models, map[string]any{
				"model_name":          name,
				"model_version":       version,
				"artifact_hash":       artifactHash,
				"artifact_path":       artifactPath,
				"feature_set_version": featureSetVersion,
				"calibration_version": calibrationVersion,
				"deployed_at":         deployedAt,
			})
		}
		out["models"] = models
		httpx.JSON(w, http.StatusOK, out)
	})

	r.Get("/metrics", metrics.Handler)
	read.Get("/audit/verify", func(w http.ResponseWriter, r *http.Request) {
		ok, msg, err := audit.Verify(r.Context(), pool)
		if err != nil {
			logger.Error("audit verify failed", map[string]any{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		status := http.StatusOK
		if !ok {
			status = http.StatusConflict
		}
		httpx.JSON(w, status, map[string]any{"ok": ok, "message": msg})
	})
	admin.Post("/models/deploy", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		var in map[string]string
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		modelName := in["model_name"]
		version := in["version"]
		if modelName == "" || version == "" {
			http.Error(w, "model_name and version required", http.StatusBadRequest)
			return
		}
		if _, err := pool.Exec(ctx, `INSERT INTO model_registry(model_name,version,state) VALUES($1,$2,'deployed') ON CONFLICT (model_name) DO UPDATE SET version=$2,state='deployed'`, modelName, version); err != nil {
			logger.Error("deploy model failed", map[string]any{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO model_deployments(model_name,model_version,artifact_hash,artifact_path,feature_set_version,calibration_version,status,created_by,approved_by,approved_at,deployed_at,change_reason) VALUES($1,$2,$3,$4,$5,$6,'deployed',$7,$7,now(),now(),$8)`, modelName, version, "inline", "/registry/"+modelName+":"+version, "v2", "", claims.Subject, "api deploy")
		if err := audit.Append(ctx, pool, claims.Subject, "model.deploy", modelName+":"+version); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		logger.Info("model deployed", map[string]any{"request_id": requestID, "actor": claims.Subject, "model_name": modelName, "version": version})
		cache.InvalidateByPrefixes(ctx, cache.ReadModelPrefixes()...)
		w.WriteHeader(http.StatusNoContent)
	})

	workflow.Post("/replay/start", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		var req struct {
			Start             time.Time `json:"start"`
			End               time.Time `json:"end"`
			ModelVersion      string    `json:"model_version"`
			FeatureSetVersion string    `json:"feature_set_version"`
			ReplayMode        string    `json:"replay_mode"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req)
		if req.Start.IsZero() {
			req.Start = time.Now().UTC().Add(-1 * time.Hour)
		}
		if req.End.IsZero() {
			req.End = time.Now().UTC()
		}
		if req.ModelVersion == "" {
			req.ModelVersion = "baseline-v1"
		}
		if req.FeatureSetVersion == "" {
			req.FeatureSetVersion = "v2"
		}
		req.ReplayMode = normalizeReplayMode(req.ReplayMode)
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO replay_jobs(requested_by,status,time_window_start,time_window_end,replay_mode,watermark_policy_id,allowed_lateness_ms,model_version,feature_set_version) VALUES($1,'queued',$2,$3,$4,'wm_v1',5000,$5,$6) RETURNING id::text`, claims.Subject, req.Start, req.End, req.ReplayMode, req.ModelVersion, req.FeatureSetVersion).Scan(&id); err != nil {
			logger.Error("start replay failed", map[string]any{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := audit.Append(ctx, pool, claims.Subject, "replay.start", id); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		metrics.IncReplayJobEvent("governance", "queued")
		logger.Info("replay job queued", map[string]any{"request_id": requestID, "actor": claims.Subject, "job_id": id, "replay_mode": req.ReplayMode})
		go func(id string) {
			_ = replay.Run(context.Background(), pool, id)
			cache.InvalidateByPrefixes(context.Background(), cache.ReadModelPrefixes()...)
		}(id)
		httpx.JSON(w, http.StatusAccepted, map[string]any{"id": id, "status": "queued", "replay_mode": req.ReplayMode})
	})

	read.Get("/models", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, loadModelRegistryRows(r.Context(), pool))
	})

	read.Get("/models/deployments", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, loadModelDeploymentRows(r.Context(), pool))
	})

	read.Get("/replay/jobs", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, loadReplayJobRows(r.Context(), pool))
	})

	read.Get("/escalation-policies", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, loadEscalationPolicyRows(r.Context(), pool))
	})

	policyAdmin.Post("/escalation-policies", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		in, err := decodeEscalationPolicyPayload(w, r)
		if err != nil {
			if !errors.Is(err, errEscalationPayload) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}
			return
		}
		var id string
		if err := pool.QueryRow(r.Context(), `INSERT INTO escalation_policies(name,severity_band,notify_channel,sla_minutes,on_call_target,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, in.Name, in.SeverityBand, in.NotifyChannel, in.SLAMinutes, in.OnCallTarget, claims.Subject).Scan(&id); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := audit.Append(r.Context(), pool, claims.Subject, "policy.create", id); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		logger.Info("escalation policy created", map[string]any{"request_id": requestID, "actor": claims.Subject, "policy_id": id})
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
	})

	policyAdmin.Put("/escalation-policies", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		in, err := decodeEscalationPolicyPayload(w, r)
		if err != nil {
			if !errors.Is(err, errEscalationPayload) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}
			return
		}
		id := strings.TrimSpace(in.ID)
		if id == "" {
			if err := pool.QueryRow(r.Context(), `INSERT INTO escalation_policies(name,severity_band,notify_channel,sla_minutes,on_call_target,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, in.Name, in.SeverityBand, in.NotifyChannel, in.SLAMinutes, in.OnCallTarget, claims.Subject).Scan(&id); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if err := audit.Append(r.Context(), pool, claims.Subject, "policy.create", id); err != nil {
				logger.Error("audit append failed", map[string]any{"error": err.Error()})
			}
			logger.Info("escalation policy created", map[string]any{"request_id": requestID, "actor": claims.Subject, "policy_id": id})
			httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
			return
		}
		ct, err := pool.Exec(r.Context(), `UPDATE escalation_policies SET name=$2,severity_band=$3,notify_channel=$4,sla_minutes=$5,on_call_target=$6 WHERE id=$1::uuid`, id, in.Name, in.SeverityBand, in.NotifyChannel, in.SLAMinutes, in.OnCallTarget)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if ct.RowsAffected() == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := audit.Append(r.Context(), pool, claims.Subject, "policy.update", id); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		logger.Info("escalation policy updated", map[string]any{"request_id": requestID, "actor": claims.Subject, "policy_id": id})
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
	})

	policyAdmin.Delete("/escalation-policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		id := chi.URLParam(r, "id")
		ct, err := pool.Exec(r.Context(), `DELETE FROM escalation_policies WHERE id=$1::uuid`, id)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if ct.RowsAffected() == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := audit.Append(r.Context(), pool, claims.Subject, "policy.delete", id); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		logger.Info("escalation policy deleted", map[string]any{"request_id": requestID, "actor": claims.Subject, "policy_id": id})
		w.WriteHeader(http.StatusNoContent)
	})

	read.Get("/on-call", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `SELECT id::text,name,current_on_call,rotation_start,rotation_end,coalesce(policy_id::text,''),created_at FROM on_call_rotations ORDER BY name`)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, name, currentOnCall, policyID string
			var rotStart, rotEnd, createdAt any
			if rows.Scan(&id, &name, &currentOnCall, &rotStart, &rotEnd, &policyID, &createdAt) == nil {
				out = append(out, map[string]any{"id": id, "name": name, "currentOnCall": currentOnCall, "rotationStart": rotStart, "rotationEnd": rotEnd, "policyId": policyID, "createdAt": createdAt})
			}
		}
		httpx.JSON(w, http.StatusOK, out)
	})
	read.Get("/current-oncall", func(w http.ResponseWriter, r *http.Request) {
		var name, currentOnCall, policyID string
		var rotStart, rotEnd, createdAt any
		err := pool.QueryRow(r.Context(), `SELECT name,current_on_call,rotation_start,rotation_end,coalesce(policy_id::text,''),created_at FROM on_call_rotations ORDER BY rotation_start DESC NULLS LAST, created_at DESC LIMIT 1`).Scan(&name, &currentOnCall, &rotStart, &rotEnd, &policyID, &createdAt)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"name": name, "currentOnCall": currentOnCall, "rotationStart": rotStart, "rotationEnd": rotEnd, "policyId": policyID, "createdAt": createdAt})
	})
	workflow.Post("/forecast/refresh", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		snapshot, err := refreshForecast(r.Context(), pool, claims.Subject)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := audit.Append(r.Context(), pool, claims.Subject, "forecast.refresh", "24h"); err != nil {
			logger.Error("audit append failed", map[string]any{"error": err.Error()})
		}
		httpx.JSON(w, http.StatusCreated, snapshot)
	})

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", map[string]any{"addr": cfg.HTTPAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("governance: server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down", nil)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := middleware.LoadAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use first allowed origin found
		var origin string
		for o := range allowedOrigins {
			origin = o
			break
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type pinger interface {
	Ping(context.Context) error
}

func readinessHandler(p pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		probeCtx, probeCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer probeCancel()
		if err := p.Ping(probeCtx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}
}

type escalationPolicyInput struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SeverityBand  string `json:"severity_band"`
	NotifyChannel string `json:"notify_channel"`
	SLAMinutes    int    `json:"sla_minutes"`
	OnCallTarget  string `json:"on_call_target"`
}

var errEscalationPayload = errors.New("invalid escalation payload")

func decodeEscalationPolicyPayload(w http.ResponseWriter, r *http.Request) (escalationPolicyInput, error) {
	var in escalationPolicyInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return escalationPolicyInput{}, errEscalationPayload
	}
	if in.Name == "" || in.SeverityBand == "" {
		http.Error(w, "name and severity_band required", http.StatusBadRequest)
		return escalationPolicyInput{}, errEscalationPayload
	}
	switch in.SeverityBand {
	case "elevated", "high", "critical":
	default:
		http.Error(w, "severity_band must be elevated, high, or critical", http.StatusBadRequest)
		return escalationPolicyInput{}, errEscalationPayload
	}
	if in.SLAMinutes <= 0 {
		in.SLAMinutes = 60
	}
	return in, nil
}

func loadModelRegistryRows(ctx context.Context, pool *pgxpool.Pool) []map[string]any {
	rows, err := pool.Query(ctx, `SELECT id,model_name,version,state,created_at FROM model_registry ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int
		var name, version, state string
		var createdAt any
		if rows.Scan(&id, &name, &version, &state, &createdAt) == nil {
			out = append(out, map[string]any{"id": id, "model_name": name, "version": version, "state": state, "created_at": createdAt})
		}
	}
	return out
}

func loadModelDeploymentRows(ctx context.Context, pool *pgxpool.Pool) []map[string]any {
	rows, err := pool.Query(ctx, `SELECT id,model_name,model_version,artifact_hash,artifact_path,feature_set_version,coalesce(calibration_version,''),status,coalesce(created_by,''),coalesce(approved_by,''),approved_at,deployed_at,change_reason,created_at FROM model_deployments ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int
		var modelName, modelVersion, artifactHash, artifactPath, featureSetVersion, calibrationVersion, status, createdBy, approvedBy, changeReason string
		var approvedAt, deployedAt, createdAt any
		if rows.Scan(&id, &modelName, &modelVersion, &artifactHash, &artifactPath, &featureSetVersion, &calibrationVersion, &status, &createdBy, &approvedBy, &approvedAt, &deployedAt, &changeReason, &createdAt) == nil {
			out = append(out, map[string]any{
				"id":                  id,
				"model_name":          modelName,
				"model_version":       modelVersion,
				"artifact_hash":       artifactHash,
				"artifact_path":       artifactPath,
				"feature_set_version": featureSetVersion,
				"calibration_version": calibrationVersion,
				"status":              status,
				"created_by":          createdBy,
				"approved_by":         approvedBy,
				"approved_at":         approvedAt,
				"deployed_at":         deployedAt,
				"change_reason":       changeReason,
				"created_at":          createdAt,
			})
		}
	}
	return out
}

func loadReplayJobRows(ctx context.Context, pool *pgxpool.Pool) []map[string]any {
	rows, err := pool.Query(ctx, `SELECT id::text,status,requested_by,requested_at,started_at,completed_at,replay_mode,model_version,feature_set_version,watermark_policy_id,allowed_lateness_ms,error_message FROM replay_jobs ORDER BY requested_at DESC LIMIT 100`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, status, requestedBy, replayMode, modelVersion, featureSetVersion, watermarkPolicy, errorMessage string
		var requestedAt, startedAt, completedAt any
		var allowedLatenessMS int
		if rows.Scan(&id, &status, &requestedBy, &requestedAt, &startedAt, &completedAt, &replayMode, &modelVersion, &featureSetVersion, &watermarkPolicy, &allowedLatenessMS, &errorMessage) == nil {
			out = append(out, map[string]any{
				"id":                id,
				"status":            status,
				"requestedBy":       requestedBy,
				"requestedAt":       requestedAt,
				"startedAt":         startedAt,
				"completedAt":       completedAt,
				"replayMode":        replayMode,
				"modelVersion":      modelVersion,
				"featureSetVersion": featureSetVersion,
				"watermarkPolicy":   watermarkPolicy,
				"allowedLatenessMs": allowedLatenessMS,
				"errorMessage":      errorMessage,
			})
		}
	}
	return out
}

func loadEscalationPolicyRows(ctx context.Context, pool *pgxpool.Pool) []map[string]any {
	rows, err := pool.Query(ctx, `SELECT id::text,name,severity_band,notify_channel,sla_minutes,on_call_target,created_by,created_at FROM escalation_policies ORDER BY severity_band,name`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, severityBand, notifyChannel, onCallTarget, createdBy string
		var slaMinutes int
		var createdAt any
		if rows.Scan(&id, &name, &severityBand, &notifyChannel, &slaMinutes, &onCallTarget, &createdBy, &createdAt) == nil {
			out = append(out, map[string]any{
				"id":            id,
				"name":          name,
				"severityBand":  severityBand,
				"notifyChannel": notifyChannel,
				"slaMinutes":    slaMinutes,
				"onCallTarget":  onCallTarget,
				"createdBy":     createdBy,
				"createdAt":     createdAt,
			})
		}
	}
	return out
}

func refreshForecast(ctx context.Context, pool *pgxpool.Pool, actor string) (map[string]any, error) {
	var velocity, dqScore, dispersion, tailRisk float64
	_ = pool.QueryRow(ctx, `
SELECT
  COALESCE(count(*)::float8 / 24.0, 0),
  COALESCE(avg(CASE WHEN trust_state = 'stable' THEN 0.9 ELSE 0.5 END), 0),
  COALESCE(stddev_pop(composite_risk), 0),
  COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY composite_risk), 0)
FROM incidents
WHERE last_activity_at > now() - interval '24 hours'`).Scan(&velocity, &dqScore, &dispersion, &tailRisk)

	rows, err := pool.Query(ctx, `
SELECT primary_symbol
FROM incidents
WHERE status IN ('open','ack')
ORDER BY composite_risk DESC, last_activity_at DESC
LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	watchlist := []string{}
	for rows.Next() {
		var symbol string
		if rows.Scan(&symbol) == nil && symbol != "" {
			watchlist = append(watchlist, symbol)
		}
	}
	base := int(velocity*4 + tailRisk*10)
	if base < 1 {
		base = 1
	}
	whyMoved := []string{
		"velocity reflects active incidents in the last 24h",
		fmt.Sprintf("dispersion %.3f widened the forecast band", dispersion),
		fmt.Sprintf("tail risk p95 %.3f raised the upside case", tailRisk),
	}
	var generatedAt time.Time
	if err := pool.QueryRow(ctx, `
INSERT INTO forecast_snapshots(window_label,incident_band_low,incident_band_base,incident_band_high,velocity,dq_score,dispersion,tail_risk_p95,watchlist_risks,why_moved,created_by)
VALUES('24h',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING generated_at`,
		maxInt(base-2, 0), base, base+3, velocity, dqScore, dispersion, tailRisk, watchlist, whyMoved, actor).Scan(&generatedAt); err != nil {
		return nil, err
	}
	return map[string]any{
		"generatedAt":        generatedAt,
		"incidentVolumeBand": map[string]int{"low": maxInt(base-2, 0), "base": base, "high": base + 3},
		"watchlistRisks":     watchlist,
		"whyMoved":           whyMoved,
		"velocity":           velocity,
		"dqScore":            dqScore,
		"dispersion":         dispersion,
		"tailRiskP95":        tailRisk,
	}, nil
}

var errNotReady = errors.New("not ready")

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func rateLimitSubjectKey(prefix string, pub *rsa.PublicKey) func(*http.Request) string {
	return func(r *http.Request) string {
		if claims, err := authz.Authenticate(r, pub); err == nil {
			return prefix + ":sub:" + claims.Subject
		}
		if authzHeader := r.Header.Get("Authorization"); len(authzHeader) > 7 {
			token := strings.TrimSpace(strings.TrimPrefix(authzHeader, "Bearer "))
			if token != "" {
				return prefix + ":authz:" + token
			}
		}
		if c, err := r.Cookie("sentinel_token"); err == nil && c.Value != "" {
			return prefix + ":cookie:" + c.Value
		}
		return prefix + ":ip:" + r.RemoteAddr
	}
}

func normalizeReplayMode(in string) string {
	switch in {
	case "as_scored", "recompute":
		return in
	default:
		return "recompute"
	}
}
