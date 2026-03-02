package main

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/twmb/franz-go/pkg/kgo"
	"sentinel/internal/audit"
	"sentinel/internal/auth"
	"sentinel/internal/authz"
	"sentinel/internal/cache"
	"sentinel/internal/config"
	"sentinel/internal/db"
	"sentinel/internal/healthcheck"
	"sentinel/internal/httpx"
	"sentinel/internal/incidents"
	"sentinel/internal/kafka"
	"sentinel/internal/logging"
	"sentinel/internal/metrics"
	"sentinel/internal/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

var subs = struct {
	sync.Mutex
	conns []chan string
}{}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8083)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.New("alerts")
	cfg := config.Load("alerts")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("alerts: invalid config: %v", err)
	}
	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("alerts: failed to connect to database: %v", err)
	}
	defer pool.Close()
	pub, err := auth.ReadPublic(cfg.JWTPublicKey)
	if err != nil {
		log.Fatalf("alerts: failed to read JWT public key: %v", err)
	}
	c, err := kafka.New(cfg.KafkaBrokers, "alerts", "derived.scores")
	if err != nil {
		log.Fatalf("alerts: failed to create kafka consumer: %v", err)
	}
	defer c.Close()
	p, err := kafka.New(cfg.KafkaBrokers, "")
	if err != nil {
		log.Fatalf("alerts: failed to create kafka producer: %v", err)
	}
	defer p.Close()

	go consumeScores(ctx, logger, pool, c, p)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)
	r.Use(metrics.HTTPMiddleware("alerts"))
	r.Use(middleware.RequireCSRFFunc(authz.SubjectFromRequest(pub)))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"service": "alerts",
			"version": "dev",
			"links": map[string]string{
				"healthz": "/healthz",
				"readyz":  "/readyz",
				"metrics": "/metrics",
				"docs":    "/docs",
			},
		})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(probeCtx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		if err := c.Ping(probeCtx); err != nil {
			http.Error(w, "kafka consumer not ready", http.StatusServiceUnavailable)
			return
		}
		if err := p.Ping(probeCtx); err != nil {
			http.Error(w, "kafka producer not ready", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	r.Get("/metrics", metrics.Handler)
	r.With(authz.RequirePerm(pub, "read")).Get("/sse/alerts", sse)

	mutation := r.With(
		middleware.RateLimit(rateLimitSubjectKey("alerts", pub), 45, time.Minute),
		authz.RequireAnyPerm(pub, "alerts:write", "*"),
	)
	read := r.With(authz.RequirePerm(pub, "read"))

	mutation.Post("/alerts/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		id := chi.URLParam(r, "id")
		start := time.Now()
		if _, err := pool.Exec(ctx, `UPDATE alerts SET status='ack',updated_at=now() WHERE id=$1`, id); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		metrics.ObserveDBQuery("alerts", "update alerts set status ack", time.Since(start))
		_ = audit.Append(ctx, pool, claims.Subject, "alert.ack", id)
		cache.InvalidateByPrefixes(ctx, cache.ReadModelPrefixes()...)
		w.WriteHeader(http.StatusNoContent)
	})

	mutation.Post("/incidents/{id}/transition", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		id := chi.URLParam(r, "id")
		var in struct {
			Command string `json:"command"`
			Owner   string `json:"owner"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.Command == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		var current string
		start := time.Now()
		if err := pool.QueryRow(ctx, `SELECT status FROM incidents WHERE id=$1`, id).Scan(&current); err != nil {
			http.Error(w, "incident not found", http.StatusNotFound)
			return
		}
		metrics.ObserveDBQuery("alerts", "select incidents status", time.Since(start))
		next, err := incidents.NextIncidentStatus(current, in.Command)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		rec := incidents.RecommendedActionForIncident(0.5, 0.7, "stable", 0, next == "suppressed")
		reason := incidents.RankReasonForIncident(0.5, 0.5, 0.7, "stable", 0, true)
		start = time.Now()
		if _, err := pool.Exec(ctx, `UPDATE incidents SET status=$2,owner_name=COALESCE(NULLIF($3,''),owner_name),updated_at=now(),last_activity_at=now(),top_driver_1=$4,top_driver_2=$5 WHERE id=$1`, id, next, in.Owner, rec, reason); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		metrics.ObserveDBQuery("alerts", "update incidents transition", time.Since(start))
		if in.Command == "escalate" {
			_, _ = notifyOnCall(ctx, pool, id, claims.Subject)
		}
		_ = audit.Append(ctx, pool, claims.Subject, "incident.transition", id+":"+next)
		httpx.JSON(w, http.StatusOK, map[string]any{"status": next, "recommended_action": rec, "rank_reason": reason})
	})

	mutation.Post("/cases", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		var in struct {
			IncidentID int64  `json:"incident_id"`
			Reason     string `json:"reason"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.IncidentID == 0 || in.Reason == "" {
			http.Error(w, "incident_id and reason required", http.StatusBadRequest)
			return
		}
		createCase(ctx, w, pool, logger, middleware.RequestIDValue(r), claims.Subject, in.IncidentID, in.Reason)
	})

	mutation.Post("/incidents/{id}/promote-case", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		var in struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.Reason == "" {
			http.Error(w, "reason required", http.StatusBadRequest)
			return
		}
		var incidentID int64
		if _, err := fmt.Sscan(chi.URLParam(r, "id"), &incidentID); err != nil || incidentID == 0 {
			http.Error(w, "invalid incident id", http.StatusBadRequest)
			return
		}
		createCase(ctx, w, pool, logger, middleware.RequestIDValue(r), claims.Subject, incidentID, in.Reason)
	})
	mutation.Post("/incidents/{id}/notify-oncall", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		incidentID := chi.URLParam(r, "id")
		if incidentID == "" {
			http.Error(w, "invalid incident id", http.StatusBadRequest)
			return
		}
		entry, err := notifyOnCall(ctx, pool, incidentID, claims.Subject)
		if err != nil {
			http.Error(w, "failed to notify", http.StatusInternalServerError)
			return
		}
		_ = audit.Append(ctx, pool, claims.Subject, "incident.notify_oncall", incidentID)
		httpx.JSON(w, http.StatusAccepted, entry)
	})

	read.Get("/cases", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(ctx, `SELECT id,incident_id,status,reason,coalesce(owner_name,''),created_at,updated_at FROM cases ORDER BY updated_at DESC LIMIT 200`)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, incidentID int64
			var status, reason, owner string
			var createdAt, updatedAt any
			if rows.Scan(&id, &incidentID, &status, &reason, &owner, &createdAt, &updatedAt) == nil {
				out = append(out, map[string]any{"id": id, "incident_id": incidentID, "status": status, "reason": reason, "owner": owner, "created_at": createdAt, "updated_at": updatedAt})
			}
		}
		httpx.JSON(w, http.StatusOK, out)
	})
	read.Get("/notifications", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(ctx, `SELECT id::text,incident_id,notification_type,target,channel,status,payload,created_by,created_at FROM incident_notifications ORDER BY created_at DESC LIMIT 100`)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, target, typ, channel, status, createdBy string
			var incidentID int64
			var payload any
			var createdAt any
			if rows.Scan(&id, &incidentID, &typ, &target, &channel, &status, &payload, &createdBy, &createdAt) == nil {
				out = append(out, map[string]any{"id": id, "incidentId": incidentID, "type": typ, "target": target, "channel": channel, "status": status, "payload": payload, "createdBy": createdBy, "createdAt": createdAt})
			}
		}
		httpx.JSON(w, http.StatusOK, out)
	})

	read.Get("/cases/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		out := map[string]any{}
		var incidentID int64
		var cid int64
		var status, reason, owner string
		var createdAt, updatedAt any
		if err := pool.QueryRow(ctx, `SELECT id,incident_id,status,reason,coalesce(owner_name,''),created_at,updated_at FROM cases WHERE id=$1`, id).Scan(&cid, &incidentID, &status, &reason, &owner, &createdAt, &updatedAt); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		out["id"] = cid
		out["status"] = status
		out["reason"] = reason
		out["owner"] = owner
		out["created_at"] = createdAt
		out["updated_at"] = updatedAt
		out["incident_id"] = incidentID
		out["notes"] = fetchRows(pool, ctx, `SELECT id,actor,note,created_at FROM case_notes WHERE case_id=$1 ORDER BY id DESC`, id, []string{"id", "actor", "note", "created_at"})
		out["evidence"] = fetchRows(pool, ctx, `SELECT id,evidence_type,reference_id,metadata,created_at FROM case_evidence WHERE case_id=$1 ORDER BY id DESC`, id, []string{"id", "evidence_type", "reference_id", "metadata", "created_at"})
		out["actions"] = fetchRows(pool, ctx, `SELECT id,actor,action,payload,created_at FROM case_actions WHERE case_id=$1 ORDER BY created_at ASC, id ASC`, id, []string{"id", "actor", "action", "payload", "created_at"})
		httpx.JSON(w, http.StatusOK, out)
	})

	mutation.Patch("/cases/{id}", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		id := chi.URLParam(r, "id")
		var in struct {
			Status string `json:"status"`
			Owner  string `json:"owner"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || (in.Status == "" && in.Owner == "") {
			http.Error(w, "status or owner required", http.StatusBadRequest)
			return
		}
		if in.Status != "" {
			var current string
			if err := pool.QueryRow(ctx, `SELECT status FROM cases WHERE id=$1`, id).Scan(&current); err != nil {
				http.Error(w, "case not found", http.StatusNotFound)
				return
			}
			if !isAllowedCaseTransition(current, in.Status) {
				http.Error(w, "invalid case status transition", http.StatusBadRequest)
				return
			}
		}
		if _, err := pool.Exec(ctx, `UPDATE cases SET status=COALESCE(NULLIF($2,''),status),owner_name=COALESCE(NULLIF($3,''),owner_name),updated_at=now() WHERE id=$1`, id, in.Status, in.Owner); err != nil {
			http.Error(w, "failed", http.StatusInternalServerError)
			return
		}
		if in.Status != "" {
			_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, id, claims.Subject, "status", map[string]any{"status": in.Status})
			_ = audit.Append(ctx, pool, claims.Subject, "case.status", id+":"+in.Status)
			metrics.IncCaseMutation("alerts", "status")
		}
		if in.Owner != "" {
			_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, id, claims.Subject, "assignment", map[string]any{"owner": in.Owner})
			_ = audit.Append(ctx, pool, claims.Subject, "case.assignment", id+":"+in.Owner)
			metrics.IncCaseMutation("alerts", "assignment")
		}
		logger.Info("case updated", map[string]any{"request_id": requestID, "actor": claims.Subject, "case_id": id, "status": in.Status, "owner": in.Owner})
		w.WriteHeader(http.StatusNoContent)
	})

	mutation.Post("/cases/{id}/notes", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		id := chi.URLParam(r, "id")
		var in struct {
			Note string `json:"note"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.Note == "" {
			http.Error(w, "note required", http.StatusBadRequest)
			return
		}
		if _, err := pool.Exec(ctx, `INSERT INTO case_notes(case_id,actor,note) VALUES($1,$2,$3)`, id, claims.Subject, in.Note); err != nil {
			http.Error(w, "failed to add note", http.StatusInternalServerError)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, id, claims.Subject, "note", map[string]any{"note": in.Note})
		_ = audit.Append(ctx, pool, claims.Subject, "case.note", id)
		metrics.IncCaseMutation("alerts", "note")
		logger.Info("case note added", map[string]any{"request_id": requestID, "actor": claims.Subject, "case_id": id})
		w.WriteHeader(http.StatusNoContent)
	})

	mutation.Post("/cases/{id}/evidence", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		id := chi.URLParam(r, "id")
		var in struct {
			EvidenceType string         `json:"evidence_type"`
			ReferenceID  string         `json:"reference_id"`
			Metadata     map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.EvidenceType == "" || in.ReferenceID == "" {
			http.Error(w, "evidence_type and reference_id required", http.StatusBadRequest)
			return
		}
		if _, err := pool.Exec(ctx, `INSERT INTO case_evidence(case_id,evidence_type,reference_id,metadata) VALUES($1,$2,$3,$4)`, id, in.EvidenceType, in.ReferenceID, in.Metadata); err != nil {
			http.Error(w, "failed", http.StatusInternalServerError)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, id, claims.Subject, "evidence", map[string]any{"reference_id": in.ReferenceID, "evidence_type": in.EvidenceType})
		_ = audit.Append(ctx, pool, claims.Subject, "case.evidence", id+":"+in.ReferenceID)
		metrics.IncCaseMutation("alerts", "evidence")
		logger.Info("case evidence attached", map[string]any{"request_id": requestID, "actor": claims.Subject, "case_id": id, "reference_id": in.ReferenceID})
		w.WriteHeader(http.StatusNoContent)
	})

	mutation.Post("/cases/{id}/disposition", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := authz.Claims(r)
		requestID := middleware.RequestIDValue(r)
		id := chi.URLParam(r, "id")
		var in struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || in.Status == "" || in.Reason == "" {
			http.Error(w, "status and reason required", http.StatusBadRequest)
			return
		}
		if _, err := pool.Exec(ctx, `UPDATE cases SET status=$2,updated_at=now() WHERE id=$1`, id, in.Status); err != nil {
			http.Error(w, "failed", http.StatusInternalServerError)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO case_notes(case_id,actor,note) VALUES($1,$2,$3)`, id, claims.Subject, "disposition: "+in.Reason)
		_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, id, claims.Subject, "disposition", map[string]any{"status": in.Status, "reason": in.Reason})
		_ = audit.Append(ctx, pool, claims.Subject, "case.disposition", id+":"+in.Status)
		metrics.IncCaseMutation("alerts", "disposition")
		logger.Info("case disposition captured", map[string]any{"request_id": requestID, "actor": claims.Subject, "case_id": id, "status": in.Status})
		w.WriteHeader(http.StatusNoContent)
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("starting server", map[string]any{"addr": cfg.HTTPAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("alerts: server error: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func consumeScores(ctx context.Context, logger logging.Logger, pool *pgxpool.Pool, c *kgo.Client, p *kgo.Client) {
	for {
		f := c.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		f.EachRecord(func(rec *kgo.Record) {
			metrics.IncKafkaConsumed("alerts", rec.Topic)
			var m map[string]any
			if err := json.Unmarshal(rec.Value, &m); err != nil {
				logger.Error("unmarshal score failed", map[string]any{"error": err.Error()})
				return
			}
			symbol, _ := m["symbol"].(string)
			sev, _ := m["severity"].(string)
			explanation, _ := m["explanation"].(string)
			score, _ := m["score"].(float64)
			escalation, _ := m["escalation_probability"].(float64)
			confidence, _ := m["confidence"].(float64)
			featureHash, _ := m["feature_snapshot_hash"].(string)
			featureSetVersion, _ := m["feature_set_version"].(string)
			modelVersion, _ := m["model_version"].(string)
			calibrationVersion, _ := m["calibration_version"].(string)
			scoringRunID, _ := m["scoring_run_id"].(string)
			artifactHash, _ := m["artifact_hash"].(string)
			modelArtifactHash, _ := m["model_artifact_hash"].(string)
			// Extract proper lineage fields from inference output
			rawAnomaly, _ := m["raw_anomaly_score"].(float64)
			normAnomaly, _ := m["normalized_anomaly_score"].(float64)
			priorityScore, _ := m["priority_score"].(float64)
			compositeRisk, _ := m["composite_risk"].(float64)
			if rawAnomaly == 0 {
				rawAnomaly = score
			}
			if normAnomaly == 0 {
				normAnomaly = score
			}
			if priorityScore == 0 {
				priorityScore = score
			}
			if compositeRisk == 0 {
				compositeRisk = score
			}
			if symbol == "" {
				return
			}
			if confidence == 0 {
				confidence = 0.8
			}
			if modelVersion == "" {
				modelVersion = "baseline-v1"
			}
			ts := scoreTS(m)
			producedAt := scoreProducedAt(m, ts)
			if featureSetVersion == "" {
				featureSetVersion = "v2"
			}
			if calibrationVersion == "" {
				calibrationVersion = "platt_v1"
			}
			idempotency := scoreIdempotencyKey(symbol, ts, score, sev, explanation)
			if scoringRunID == "" {
				scoringRunID = idempotency
			}
			if artifactHash == "" {
				artifactHash = idempotency
			}
			if modelArtifactHash == "" {
				modelArtifactHash = modelVersion
			}
			var scoreID int64
			start := time.Now()
			if err := pool.QueryRow(ctx, `INSERT INTO scores(idempotency_key,symbol,ts,score,severity,explanation,raw_anomaly_score,normalized_anomaly_score,escalation_probability,priority_score,composite_risk,feature_snapshot_hash,feature_set_version,model_version,calibration_version,scoring_run_id,produced_at,artifact_hash,model_artifact_hash,explanation_payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,COALESCE(NULLIF($16,''), gen_random_uuid()::text)::uuid,$17,$18,$19,$20) ON CONFLICT (idempotency_key,ts) DO UPDATE SET score=EXCLUDED.score,normalized_anomaly_score=EXCLUDED.normalized_anomaly_score,composite_risk=EXCLUDED.composite_risk,feature_snapshot_hash=EXCLUDED.feature_snapshot_hash,feature_set_version=EXCLUDED.feature_set_version,model_version=EXCLUDED.model_version,calibration_version=EXCLUDED.calibration_version,scoring_run_id=EXCLUDED.scoring_run_id,produced_at=EXCLUDED.produced_at,artifact_hash=EXCLUDED.artifact_hash,model_artifact_hash=EXCLUDED.model_artifact_hash RETURNING id`, idempotency, symbol, ts, score, sev, explanation, rawAnomaly, normAnomaly, escalation, priorityScore, compositeRisk, featureHash, featureSetVersion, modelVersion, calibrationVersion, scoringRunID, producedAt, artifactHash, modelArtifactHash, map[string]any{"explanation": explanation, "top_drivers": m["top_drivers"]}).Scan(&scoreID); err != nil {
				return
			}
			metrics.ObserveDBQuery("alerts", "upsert scores", time.Since(start))
			if sev == "" || sev == "stable" || sev == "low" {
				return
			}
			start = time.Now()
			incidentID, err := incidents.FindOrCreateOpenIncident(ctx, pool, symbol, score, escalation, confidence, featureHash, modelVersion, ts)
			if err != nil {
				return
			}
			metrics.ObserveDBQuery("alerts", "upsert incidents", time.Since(start))
			var alertID int64
			start = time.Now()
			if err := pool.QueryRow(ctx, `INSERT INTO alerts(score_id,incident_id,symbol,status,justification) VALUES($1,$2,$3,'open','auto') RETURNING id`, scoreID, incidentID, symbol).Scan(&alertID); err != nil {
				return
			}
			metrics.ObserveDBQuery("alerts", "insert alerts", time.Since(start))
			_, _ = pool.Exec(ctx, `INSERT INTO incident_alert_links(incident_id,alert_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, incidentID, alertID)
			if err := kafka.ProduceJSON(ctx, p, "alerts.created", symbol, m); err == nil {
				metrics.IncKafkaProduced("alerts", "alerts.created")
			}
			_ = audit.Append(ctx, pool, "system", "alert.create", symbol)
			broadcast(fmt.Sprintf("event: alert\ndata: %s\n\n", string(rec.Value)))
		})
	}
}

func createCase(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, logger logging.Logger, requestID, actor string, incidentID int64, reason string) {
	var incidentStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM incidents WHERE id=$1`, incidentID).Scan(&incidentStatus); err != nil {
		http.Error(w, "incident not found", http.StatusNotFound)
		return
	}
	if !incidents.CanPromoteToCase(incidentStatus) {
		http.Error(w, "incident cannot be promoted", http.StatusConflict)
		return
	}
	var existing int64
	if err := pool.QueryRow(ctx, `SELECT id FROM cases WHERE incident_id=$1 AND status IN ('open','investigating','escalated') ORDER BY id DESC LIMIT 1`, incidentID).Scan(&existing); err == nil {
		httpx.JSON(w, http.StatusConflict, map[string]any{"error": "duplicate active case", "existing_case_id": existing})
		return
	}
	var caseID int64
	if err := pool.QueryRow(ctx, `INSERT INTO cases(incident_id,reason,owner_name) VALUES($1,$2,$3) RETURNING id`, incidentID, reason, actor).Scan(&caseID); err != nil {
		http.Error(w, "failed to create case", http.StatusInternalServerError)
		return
	}
	_, _ = pool.Exec(ctx, `INSERT INTO case_actions(case_id,actor,action,payload) VALUES($1,$2,$3,$4)`, caseID, actor, "create", map[string]any{"incident_id": incidentID, "reason": reason})
	_ = audit.Append(ctx, pool, actor, "case.create", fmt.Sprintf("%d", caseID))
	metrics.IncCaseMutation("alerts", "create")
	logger.Info("case created", map[string]any{"request_id": requestID, "actor": actor, "case_id": caseID, "incident_id": incidentID})
	httpx.JSON(w, http.StatusCreated, map[string]any{"case_id": caseID, "reused": false})
}

func notifyOnCall(ctx context.Context, pool *pgxpool.Pool, incidentID, actor string) (map[string]any, error) {
	var target, policyID string
	err := pool.QueryRow(ctx, `SELECT current_on_call, coalesce(policy_id::text,'') FROM on_call_rotations ORDER BY rotation_start DESC NULLS LAST, created_at DESC LIMIT 1`).Scan(&target, &policyID)
	if err != nil {
		target = "unassigned"
	}
	payload := map[string]any{
		"incident_id": incidentID,
		"message":     "Sentinel notify-on-call stub",
		"policy_id":   policyID,
	}
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO incident_notifications(incident_id,notification_type,target,channel,status,payload,created_by) VALUES($1::bigint,'on_call',$2,'webhook_stub','sent',$3,$4) RETURNING id::text`, incidentID, target, payload, actor).Scan(&id); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "incidentId": incidentID, "target": target, "channel": "webhook_stub", "status": "sent", "payload": payload}, nil
}

func fetchRows(pool *pgxpool.Pool, ctx context.Context, q, id string, cols []string) []map[string]any {
	rows, err := pool.Query(ctx, q, id)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if rows.Scan(ptrs...) == nil {
			m := map[string]any{}
			for i, c := range cols {
				m[c] = vals[i]
			}
			out = append(out, m)
		}
	}
	return out
}

func isAllowedCaseTransition(current, next string) bool {
	allowed := map[string]map[string]bool{
		"open":          {"investigating": true, "escalated": true, "closed": true},
		"investigating": {"escalated": true, "closed": true, "open": true},
		"escalated":     {"investigating": true, "closed": true},
		"closed":        {},
	}
	if current == next {
		return true
	}
	return allowed[current][next]
}

func scoreTS(m map[string]any) time.Time {
	if f, ok := m["ts"].(float64); ok && f > 0 {
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		return time.Unix(sec, nsec).UTC()
	}
	if s, ok := m["ts"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func scoreProducedAt(m map[string]any, fallback time.Time) time.Time {
	if s, ok := m["produced_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UTC()
		}
	}
	return fallback
}

func scoreIdempotencyKey(symbol string, ts time.Time, score float64, severity, explanation string) string {
	raw := fmt.Sprintf("%s|%s|%.6f|%s|%s", symbol, ts.Format(time.RFC3339Nano), score, severity, explanation)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func rateLimitSubjectKey(prefix string, pub *rsa.PublicKey) func(*http.Request) string {
	return func(r *http.Request) string {
		claims, err := authz.Authenticate(r, pub)
		if err == nil {
			return prefix + ":sub:" + claims.Subject
		}
		if authzHeader := r.Header.Get("Authorization"); authzHeader != "" {
			return prefix + ":authz:" + authzHeader
		}
		if c, err := r.Cookie("sentinel_token"); err == nil && c.Value != "" {
			return prefix + ":cookie:" + c.Value
		}
		return prefix + ":ip:" + r.RemoteAddr
	}
}

func sse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan string, 16)
	subs.Lock()
	subs.conns = append(subs.conns, ch)
	subs.Unlock()
	ctx := r.Context()
	defer func() {
		subs.Lock()
		for i, c := range subs.conns {
			if c == ch {
				subs.conns = append(subs.conns[:i], subs.conns[i+1:]...)
				break
			}
		}
		subs.Unlock()
		close(ch)
	}()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case m := <-ch:
			if _, err := w.Write([]byte(m)); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func broadcast(msg string) {
	subs.Lock()
	defer subs.Unlock()
	for _, c := range subs.conns {
		select {
		case c <- msg:
		default:
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := middleware.LoadAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use first allowed origin found; for SSE streaming, this is acceptable
		var origin string
		for o := range allowedOrigins {
			origin = o
			break
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
