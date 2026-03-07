package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
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
	qrm "sentinel/internal/query"
)

const (
	ttlQueue     = 3 * time.Second
	ttlSummary   = 5 * time.Second
	ttlWorldMap  = 10 * time.Second
	ttlExecutive = 10 * time.Second
)

type seedStatusCheck struct {
	Name  string
	Query string
}

var seedStatusChecks = []seedStatusCheck{
	{Name: "raw_ticks", Query: "SELECT count(*) FROM raw_ticks"},
	{Name: "candles", Query: "SELECT count(*) FROM candles"},
	{Name: "features", Query: "SELECT count(*) FROM features"},
	{Name: "scores", Query: "SELECT count(*) FROM scores"},
	{Name: "alerts", Query: "SELECT count(*) FROM alerts"},
	{Name: "incidents", Query: "SELECT count(*) FROM incidents"},
	{Name: "cases", Query: "SELECT count(*) FROM cases"},
}

func seedStatusHandler(counter func(context.Context, string) (int64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out := map[string]int64{}
		for _, c := range seedStatusChecks {
			n, err := counter(r.Context(), c.Query)
			if err != nil {
				http.Error(w, "internal", http.StatusInternalServerError)
				return
			}
			out[c.Name] = n
		}
		httpx.JSON(w, http.StatusOK, out)
	}
}

type filters struct {
	Window      string
	From        time.Time
	To          time.Time
	Q           string
	CountryCode string
	Region      string
	Sector      string
	Industry    string
	Venue       string
	Symbol      string
	Locale      string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8085)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.New("query")
	cfg := config.Load("query")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("query: invalid config: %v", err)
	}
	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("query: failed db: %v", err)
	}
	defer pool.Close()
	pub, err := auth.ReadPublic(cfg.JWTPublicKey)
	if err != nil {
		log.Fatalf("query: failed read key: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)
	r.Use(metrics.HTTPMiddleware("query"))
	r.Use(middleware.RequireCSRFFunc(authz.SubjectFromRequest(pub)))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"service": "query",
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
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", 503)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/metrics", metrics.Handler)
	debugHandler := seedStatusHandler(func(ctx context.Context, query string) (int64, error) {
		var n int64
		if err := pool.QueryRow(ctx, query).Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	})
	r.With(authz.RequirePerm(pub, "read")).Get("/debug/seed-status", debugHandler)
	r.With(authz.RequirePerm(pub, "admin")).Post("/dev/backfill", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.DemoMode {
			http.NotFound(w, r)
			return
		}
		years := parseFeedLimit(r.URL.Query().Get("years"), 20, 20)
		if err := qrm.BackfillHistory(r.Context(), pool, years); err != nil {
			http.Error(w, "backfill failed", http.StatusInternalServerError)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok", "years": years})
	})

	r.Group(func(pr chi.Router) {
		pr.Use(authz.RequirePerm(pub, "read"))
		pr.Get("/queue", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			claims, _ := authz.Claims(r)
			key := cache.QueueKey(claims.Role, f.Region, f.CountryCode, f.Sector, f.Industry, f.Venue, f.Symbol, f.Locale, f.Window+":"+f.Q)
			out, err := cache.GetOrLoadJSON(r.Context(), key, ttlQueue, func() ([]qrm.QueueRow, error) {
				return qrm.LoadQueue(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
			})
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			metrics.ObserveRowsReturned("query", "queue", len(out))
			httpx.JSON(w, 200, out)
		})
		pr.Get("/command-center", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			claims, _ := authz.Claims(r)
			key := cache.CommandCenterKey(claims.Role, f.Region, f.CountryCode, f.Sector, f.Industry, f.Venue, f.Symbol, f.Locale, f.Window+":"+f.Q)
			out, err := cache.GetOrLoadJSON(r.Context(), key, ttlSummary, func() (map[string]any, error) {
				return qrm.LoadCommandCenter(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
			})
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			httpx.JSON(w, 200, out)
		})
		pr.Get("/feed", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			limit := parseFeedLimit(r.URL.Query().Get("limit"), 50, 200)
			cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
			mode := r.URL.Query().Get("mode")
			switch mode {
			case "trending", "for_you":
			default:
				mode = ""
			}
			page, err := qrm.LoadRiskFeed(r.Context(), pool, qrm.QueueFilters{
				Window:      f.Window,
				From:        f.From,
				To:          f.To,
				Q:           f.Q,
				CountryCode: f.CountryCode,
				Region:      f.Region,
				Sector:      f.Sector,
				Industry:    f.Industry,
				Venue:       f.Venue,
				Symbol:      f.Symbol,
				Locale:      f.Locale,
			}, limit, cursor, mode)
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			metrics.ObserveRowsReturned("query", "feed", len(page.Items))
			httpx.JSON(w, 200, map[string]any{
				"items":             page.Items,
				"next_cursor":       page.NextCursor,
				"server_time":       page.ServerTime,
				"history":           page.History,
				"history_limited":   page.HistoryLimited,
				"window":            f.Window,
				"limit":             limit,
				"mode":              mode,
				"dev_backfill_path": "/api/proxy/query/dev/backfill?years=20",
			})
		})
		pr.Get("/search-suggest", func(w http.ResponseWriter, r *http.Request) {
			q := strings.TrimSpace(r.URL.Query().Get("q"))
			kindFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("k")))
			limit := parseFeedLimit(r.URL.Query().Get("limit"), 12, 40)
			if q == "" {
				httpx.JSON(w, 200, map[string]any{"items": []any{}})
				return
			}
			like := "%" + strings.ToLower(q) + "%"
			items := []map[string]string{}
			add := func(kind, value string) {
				v := strings.TrimSpace(value)
				if v == "" {
					return
				}
				kk := kind
				if len(kk) > 1 {
					kk = strings.ToUpper(kk[:1]) + kk[1:]
				} else {
					kk = strings.ToUpper(kk)
				}
				label := fmt.Sprintf("%s | %s", kk, v)
				for _, it := range items {
					if it["label"] == label {
						return
					}
				}
				items = append(items, map[string]string{"kind": kind, "value": v, "label": label})
			}

			loadStrings := func(query, kind string) {
				rows, err := pool.Query(r.Context(), query, like, limit)
				if err != nil {
					return
				}
				defer rows.Close()
				for rows.Next() {
					var v string
					if rows.Scan(&v) == nil {
						add(kind, v)
						if len(items) >= limit {
							return
						}
					}
				}
			}

			loadKind := func(kind string) {
				if kindFilter != "" && kindFilter != kind {
					return
				}
				switch kind {
				case "symbol":
					loadStrings(`SELECT DISTINCT primary_symbol FROM incidents WHERE lower(primary_symbol) LIKE $1 ORDER BY 1 LIMIT $2`, "symbol")
				case "country":
					loadStrings(`SELECT DISTINCT country_code FROM instrument_metadata WHERE lower(country_code) LIKE $1 ORDER BY 1 LIMIT $2`, "country")
				case "region":
					loadStrings(`SELECT DISTINCT region FROM instrument_metadata WHERE lower(region) LIKE $1 ORDER BY 1 LIMIT $2`, "region")
				case "sector":
					loadStrings(`SELECT DISTINCT sector FROM instrument_metadata WHERE lower(sector) LIKE $1 ORDER BY 1 LIMIT $2`, "sector")
				case "industry":
					loadStrings(`SELECT DISTINCT industry FROM instrument_metadata WHERE lower(industry) LIKE $1 ORDER BY 1 LIMIT $2`, "industry")
				case "venue":
					loadStrings(`SELECT DISTINCT venue FROM instrument_metadata WHERE lower(venue) LIKE $1 ORDER BY 1 LIMIT $2`, "venue")
				}
			}

			for _, kind := range []string{"symbol", "country", "region", "sector", "industry", "venue"} {
				if len(items) >= limit {
					break
				}
				loadKind(kind)
			}

			// Fallback suggestions for empty/demo datasets.
			if len(items) == 0 {
				demo := []map[string]string{
					{"kind": "symbol", "value": "BTC/USD"},
					{"kind": "symbol", "value": "AAPL"},
					{"kind": "symbol", "value": "TSLA"},
					{"kind": "sector", "value": "Technology"},
					{"kind": "sector", "value": "Energy"},
					{"kind": "country", "value": "US"},
					{"kind": "country", "value": "GB"},
				}
				for _, d := range demo {
					if kindFilter != "" && kindFilter != d["kind"] {
						continue
					}
					if strings.Contains(strings.ToLower(d["value"]), strings.ToLower(q)) {
						add(d["kind"], d["value"])
					}
					if len(items) >= limit {
						break
					}
				}
			}

			httpx.JSON(w, 200, map[string]any{"items": items})
		})
		pr.Get("/trust", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			key := cache.TrustKey(f.Region, f.CountryCode, f.Sector, f.Industry, f.Venue, f.Symbol, f.Locale, f.Window+":"+f.Q)
			out, err := cache.GetOrLoadJSON(r.Context(), key, ttlSummary, func() (map[string]any, error) {
				return qrm.LoadTrust(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
			})
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			httpx.JSON(w, 200, out)
		})
		pr.Get("/world-map", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			key := cache.WorldMapKey(f.Region, f.CountryCode, f.Sector, f.Industry, f.Venue, f.Symbol, f.Locale, f.Window+":"+f.Q)
			worldMap, err := cache.GetOrLoadJSON(r.Context(), key, ttlWorldMap, func() (qrm.WorldMapSummary, error) {
				return qrm.LoadWorldMapSummary(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
			})
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			metrics.ObserveRowsReturned("query", "world_map", len(worldMap.Countries))
			httpx.JSON(w, 200, map[string]any{"countries": worldMap.Countries, "geoEnriched": worldMap.GeoEnriched, "missingGeoCount": worldMap.MissingGeoCount, "timeWindow": f.Window})
		})
		pr.Get("/executive-summary", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			key := cache.ExecutiveKey(f.Region, f.CountryCode, f.Sector, f.Industry, f.Venue, f.Symbol, f.Locale, f.Window+":"+f.Q)
			out, err := cache.GetOrLoadJSON(r.Context(), key, ttlExecutive, func() (map[string]any, error) {
				topRisks, err := qrm.LoadQueue(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
				if err != nil {
					return nil, err
				}
				countryConcentration, err := qrm.LoadWorldMap(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
				if err != nil {
					return nil, err
				}
				trust, _ := qrm.LoadTrust(r.Context(), pool, qrm.QueueFilters{Window: f.Window, From: f.From, To: f.To, Q: f.Q, CountryCode: f.CountryCode, Region: f.Region, Sector: f.Sector, Industry: f.Industry, Venue: f.Venue, Symbol: f.Symbol, Locale: f.Locale})
				top := make([]map[string]any, 0, 5)
				for i, row := range topRisks {
					if i >= 5 {
						break
					}
					top = append(top, map[string]any{"id": row.ID, "symbol": row.Symbol, "priorityScore": row.PriorityScore, "compositeRisk": row.CompositeRisk, "severityBand": row.SeverityBand})
				}
				industryConcentration := []map[string]any{}
				industryWhere, industryArgs := buildExecutiveIndustryWhere(f)
				addIndustry := func(col, val string) {
					if val == "" {
						return
					}
					industryArgs = append(industryArgs, val)
					industryWhere += fmt.Sprintf(" AND %s=$%d", col, len(industryArgs))
				}
				addIndustry("m.region", f.Region)
				addIndustry("m.country_code", f.CountryCode)
				addIndustry("m.industry", f.Industry)
				addIndustry("m.sector", f.Sector)
				addIndustry("m.venue", f.Venue)
				addIndustry("i.primary_symbol", f.Symbol)
				ir, err := pool.Query(r.Context(), `SELECT coalesce(m.industry,'Unknown'),count(i.id) FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE `+industryWhere+` GROUP BY 1 ORDER BY 2 DESC LIMIT 8`, industryArgs...)
				if err == nil {
					defer ir.Close()
					for ir.Next() {
						var industry string
						var count int
						if ir.Scan(&industry, &count) == nil {
							industryConcentration = append(industryConcentration, map[string]any{"industry": industry, "incidentCount": count})
						}
					}
				}
				riskMemo := "No material change."
				if len(top) > 0 {
					riskMemo = "Top risk concentration is increasing in the leading instruments for the selected time window."
				}
				return map[string]any{"topRisks": top, "countryConcentration": countryConcentration, "industryConcentration": industryConcentration, "trustSummary": trust["trustSummary"], "riskMemo": riskMemo}, nil
			})
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			httpx.JSON(w, 200, out)
		})
		pr.Get("/governance/summary", func(w http.ResponseWriter, r *http.Request) {
			lineage := []map[string]any{}
			rows, err := pool.Query(r.Context(), `SELECT model_name,version,state,created_at FROM model_registry ORDER BY created_at DESC LIMIT 20`)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var n, v, s string
					var t any
					if rows.Scan(&n, &v, &s, &t) == nil {
						lineage = append(lineage, map[string]any{"model": n, "version": v, "state": s, "createdAt": t})
					}
				}
			}
			replays := []map[string]any{}
			rp, err := pool.Query(r.Context(), `SELECT rj.id::text,rj.status,rj.requested_at,rj.started_at,rj.completed_at,rj.replay_mode,rj.model_version,rj.feature_set_version,rj.watermark_policy_id,rj.allowed_lateness_ms,rr.diff_summary
FROM replay_jobs rj
LEFT JOIN replay_runs rr ON rr.id = rj.id::text
ORDER BY rj.requested_at DESC
LIMIT 25`)
			if err == nil {
				defer rp.Close()
				for rp.Next() {
					var id, st, rm, mv, fv, wp string
					var req, stt, ct any
					var late int
					var raw any
					if rp.Scan(&id, &st, &req, &stt, &ct, &rm, &mv, &fv, &wp, &late, &raw) == nil {
						summary := coerceReplaySummary(raw)
						replays = append(replays, map[string]any{
							"id":                id,
							"status":            st,
							"requestedAt":       req,
							"startedAt":         stt,
							"completedAt":       ct,
							"replayMode":        rm,
							"modelVersion":      mv,
							"featureSetVersion": fv,
							"watermarkPolicy":   wp,
							"allowedLatenessMs": late,
							"parityStatus":      replaySummaryValue(summary, "parity_status", "UNKNOWN"),
							"matchedCount":      replaySummaryValue(summary, "matched_count", 0),
							"mismatchedCount":   replaySummaryValue(summary, "mismatched_count", 0),
							"maxScoreDelta":     replaySummaryValue(summary, "max_score_delta", 0),
						})
					}
				}
			}
			httpx.JSON(w, 200, map[string]any{"modelLineage": lineage, "replayJobs": replays, "thresholdChanges": []any{}})
		})
		pr.Get("/replay/{job}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "job")
			var status, replayMode, mv, fv, wp string
			var late int
			var s, e, requested, started, completed any
			if err := pool.QueryRow(r.Context(), `SELECT status,replay_mode,time_window_start,time_window_end,requested_at,started_at,completed_at,model_version,feature_set_version,watermark_policy_id,allowed_lateness_ms FROM replay_jobs WHERE id=$1`, id).Scan(&status, &replayMode, &s, &e, &requested, &started, &completed, &mv, &fv, &wp, &late); err != nil {
				http.Error(w, "not found", 404)
				return
			}
			var result any = map[string]any{}
			_ = pool.QueryRow(r.Context(), `SELECT diff_summary FROM replay_runs WHERE id=$1`, id).Scan(&result)
			summary := coerceReplaySummary(result)
			selectedMode := normalizeReplayViewMode(r.URL.Query().Get("mode"))
			partial := status != "completed"
			resp := map[string]any{"id": id, "status": status, "replayMode": replayMode, "selectedMode": selectedMode, "timeWindowStart": s, "timeWindowEnd": e, "requestedAt": requested, "startedAt": started, "completedAt": completed, "modelVersion": mv, "featureSetVersion": fv, "watermarkPolicy": wp, "allowedLatenessMs": late, "result": summary, "partial": partial, "limitations": []string{"replay recomputes scores from raw ticks and compares them to stored scores for the same window"}}
			if selectedMode == "as_scored" {
				resp["selectedStats"] = summary["as_scored"]
			} else {
				resp["selectedStats"] = summary["recomputed"]
			}
			httpx.JSON(w, 200, resp)
		})
		pr.Get("/replay/{job}/summary", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "job")
			var status string
			if err := pool.QueryRow(r.Context(), `SELECT status FROM replay_jobs WHERE id=$1`, id).Scan(&status); err != nil {
				http.Error(w, "not found", 404)
				return
			}
			var raw any
			_ = pool.QueryRow(r.Context(), `SELECT diff_summary FROM replay_runs WHERE id=$1`, id).Scan(&raw)
			httpx.JSON(w, 200, map[string]any{
				"id":      id,
				"status":  status,
				"summary": coerceReplaySummary(raw),
			})
		})
		pr.Get("/replay/{job}/brief", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "job")
			var status, replayMode, mv, fv, wp string
			var late int
			var s, e, requested, started, completed any
			if err := pool.QueryRow(r.Context(), `SELECT status,replay_mode,time_window_start,time_window_end,requested_at,started_at,completed_at,model_version,feature_set_version,watermark_policy_id,allowed_lateness_ms FROM replay_jobs WHERE id=$1`, id).Scan(&status, &replayMode, &s, &e, &requested, &started, &completed, &mv, &fv, &wp, &late); err != nil {
				http.Error(w, "not found", 404)
				return
			}
			var raw any
			_ = pool.QueryRow(r.Context(), `SELECT diff_summary FROM replay_runs WHERE id=$1`, id).Scan(&raw)
			job := map[string]any{
				"id":                id,
				"status":            status,
				"replayMode":        replayMode,
				"timeWindowStart":   s,
				"timeWindowEnd":     e,
				"requestedAt":       requested,
				"startedAt":         started,
				"completedAt":       completed,
				"modelVersion":      mv,
				"featureSetVersion": fv,
				"watermarkPolicy":   wp,
				"allowedLatenessMs": late,
			}
			payload := replayBriefPayload(job, coerceReplaySummary(raw))
			if strings.EqualFold(r.URL.Query().Get("format"), "md") {
				w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
				_, _ = fmt.Fprintf(w, "# Replay %s brief\n\n- Window: %v -> %v\n- Model version: %v\n- Feature set version: %v\n- Replay mode: %v\n- Parity: %v\n- Determinism: %v\n- Matched / mismatched: %v / %v\n- Max score delta: %v\n- Pinned evidence: %v\n- Requested: %v\n- Started: %v\n- Completed: %v\n- Exported: %v\n", id, s, e, mv, fv, replayMode, payload["parityStatus"], payload["determinismStatus"], payload["matchedCount"], payload["mismatchedCount"], payload["maxScoreDelta"], payload["pinnedEvidence"], requested, started, completed, payload["timestamps"].(map[string]any)["exportedAt"])
				return
			}
			httpx.JSON(w, 200, payload)
		})
		pr.Get("/incident/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var out = map[string]any{"id": id}
			var status, sym, sev, mv string
			var p, c float64
			if err := pool.QueryRow(r.Context(), `SELECT status,primary_symbol,severity_band,coalesce(priority_score,0),coalesce(composite_risk,0),coalesce(model_version,'') FROM incidents WHERE id=$1`, id).Scan(&status, &sym, &sev, &p, &c, &mv); err == nil {
				out["status"] = status
				out["symbol"] = sym
				out["severity"] = sev
				out["score_header"] = map[string]any{"priority": p, "composite": c}
				out["model_version"] = mv
			}
			httpx.JSON(w, 200, out)
		})
		pr.Get("/incident/{id}/brief", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var status, sym, sev, modelVersion, driver1, driver2, rec string
			var priority, composite, escalation float64
			err := pool.QueryRow(r.Context(), `SELECT status,primary_symbol,severity_band,coalesce(model_version,''),coalesce(top_driver_1,''),coalesce(top_driver_2,''),coalesce(priority_score,0),coalesce(composite_risk,0),coalesce(escalation_probability,0),coalesce(top_driver_1,'watch') FROM incidents WHERE id=$1`, id).Scan(&status, &sym, &sev, &modelVersion, &driver1, &driver2, &priority, &composite, &escalation, &rec)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			payload := map[string]any{
				"incidentId":        id,
				"whatHappened":      fmt.Sprintf("%s moved into %s risk with %.2f priority", sym, sev, priority),
				"impactedSymbols":   []string{sym},
				"topDrivers":        []string{driver1, driver2},
				"recommendedAction": rec,
				"status":            status,
				"severityBand":      sev,
				"modelVersion":      modelVersion,
				"compositeRisk":     composite,
				"escalationProb":    escalation,
			}
			if strings.EqualFold(r.URL.Query().Get("format"), "md") {
				w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
				_, _ = fmt.Fprintf(w, "# Incident %s brief\n\n- Symbol: %s\n- Status: %s\n- Severity: %s\n- Priority: %.2f\n- Composite risk: %.2f\n- Escalation probability: %.2f\n- Top drivers: %s, %s\n- Recommended action: %s\n", id, sym, status, sev, priority, composite, escalation, driver1, driver2, rec)
				return
			}
			httpx.JSON(w, 200, payload)
		})
		pr.Get("/cases", func(w http.ResponseWriter, r *http.Request) {
			rows, err := pool.Query(r.Context(), `SELECT id,incident_id,status,reason,coalesce(owner_name,''),created_at,updated_at FROM cases ORDER BY updated_at DESC LIMIT 200`)
			if err != nil {
				httpx.JSON(w, 200, []any{})
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
			httpx.JSON(w, 200, out)
		})
		pr.Get("/case/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var incidentID int64
			var cid int64
			var status, reason, owner string
			var createdAt, updatedAt any
			if err := pool.QueryRow(r.Context(), `SELECT id,incident_id,status,reason,coalesce(owner_name,''),created_at,updated_at FROM cases WHERE id=$1`, id).Scan(&cid, &incidentID, &status, &reason, &owner, &createdAt, &updatedAt); err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			httpx.JSON(w, 200, map[string]any{"id": cid, "status": status, "reason": reason, "owner": owner, "created_at": createdAt, "updated_at": updatedAt, "incident_id": incidentID})
		})
		pr.Get("/scores", func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			symbol := firstNonEmpty(strings.TrimSpace(r.URL.Query().Get("symbol")), f.Symbol)
			window := f.Window
			maxPoints := parseFeedLimit(r.URL.Query().Get("maxPoints"), 300, 1000)
			series, err := qrm.LoadScoreSeries(r.Context(), pool, symbol, window, maxPoints, f.From, f.To)
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			metrics.ObserveRowsReturned("query", "scores", len(series))
			httpx.JSON(w, 200, map[string]any{
				"series":    series,
				"symbol":    symbol,
				"window":    window,
				"maxPoints": maxPoints,
			})
		})
		pr.Get("/candles", func(w http.ResponseWriter, r *http.Request) {
			symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
			window := normalizeWindow(r.URL.Query().Get("window"))
			res := strings.TrimSpace(r.URL.Query().Get("res"))
			if window == "" {
				window = "24h"
			}
			if res == "" {
				res = "1m"
			}
			maxPoints := parseFeedLimit(r.URL.Query().Get("maxPoints"), 300, 1000)
			series, err := qrm.LoadCandleSeries(r.Context(), pool, symbol, window, res, maxPoints)
			if err != nil {
				http.Error(w, "internal", 500)
				return
			}
			httpx.JSON(w, 200, map[string]any{
				"series":    series,
				"symbol":    symbol,
				"window":    window,
				"res":       res,
				"maxPoints": maxPoints,
			})
		})
		pr.Get("/forecast", func(w http.ResponseWriter, r *http.Request) {
			var generatedAt any
			var incidentLow, incidentBase, incidentHigh int
			var watchlist []string
			var explain []string
			err := pool.QueryRow(r.Context(), `SELECT generated_at,incident_band_low,incident_band_base,incident_band_high,watchlist_risks,why_moved FROM forecast_snapshots ORDER BY generated_at DESC LIMIT 1`).Scan(&generatedAt, &incidentLow, &incidentBase, &incidentHigh, &watchlist, &explain)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			httpx.JSON(w, 200, map[string]any{
				"generatedAt":        generatedAt,
				"incidentVolumeBand": map[string]int{"low": incidentLow, "base": incidentBase, "high": incidentHigh},
				"watchlistRisks":     watchlist,
				"whyMoved":           explain,
			})
		})
		pr.Get("/stream/queue", sseStream(func(ctx context.Context) any {
			rows, _ := qrm.LoadQueue(ctx, pool, qrm.QueueFilters{Window: "24h"})
			inc := map[string]any{"id": 0}
			if len(rows) > 0 {
				inc = map[string]any{"id": rows[0].ID, "symbol": rows[0].Symbol, "priorityScore": rows[0].PriorityScore, "severityBand": rows[0].SeverityBand, "recommendedAction": rows[0].RecommendedAction}
			}
			return map[string]any{"type": "upsert", "incident": inc}
		}, "queue_patch"))
		pr.Get("/stream/command-center", sseStream(func(ctx context.Context) any {
			cc, _ := qrm.LoadCommandCenter(ctx, pool, qrm.QueueFilters{Window: "24h"})
			return cc
		}, "command_center_patch"))
		pr.Get("/stream/trust", sseStream(func(ctx context.Context) any {
			t, _ := qrm.LoadTrust(ctx, pool, qrm.QueueFilters{Window: "24h"})
			return t
		}, "trust_patch"))
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("query server: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	_ = srv.Shutdown(context.Background())
	logger.Info("query shutdown", nil)
}

func sseStream(payload func(context.Context) any, event string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "unsupported", 500)
			return
		}
		_, _ = w.Write([]byte(": connected\n\n"))
		flusher.Flush()
		tk := time.NewTicker(5 * time.Second)
		defer tk.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-tk.C:
				b, _ := json.Marshal(payload(r.Context()))
				_, _ = w.Write([]byte(": heartbeat\n"))
				_, _ = w.Write([]byte("event: " + event + "\n"))
				_, _ = w.Write([]byte("data: " + string(b) + "\n\n"))
				flusher.Flush()
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseFilters(r *http.Request) (filters, error) {
	q := r.URL.Query()
	tw := normalizeWindow(q.Get("window"))
	if tw == "" {
		tw = normalizeWindow(q.Get("time_window"))
	}
	startRaw := firstNonEmpty(q.Get("start"), q.Get("from"))
	endRaw := firstNonEmpty(q.Get("end"), q.Get("to"))
	if tw != "" && (startRaw != "" || endRaw != "") {
		return filters{}, fmt.Errorf("window cannot be combined with start/end")
	}
	if startRaw != "" && endRaw == "" {
		return filters{}, fmt.Errorf("start requires end")
	}
	if endRaw != "" && startRaw == "" {
		return filters{}, fmt.Errorf("end requires start")
	}

	if tw == "" {
		tw = "24h"
	}
	var from time.Time
	var to time.Time
	if startRaw != "" && endRaw != "" {
		var err error
		from, err = time.Parse(time.RFC3339, startRaw)
		if err != nil {
			return filters{}, fmt.Errorf("invalid start")
		}
		to, err = time.Parse(time.RFC3339, endRaw)
		if err != nil {
			return filters{}, fmt.Errorf("invalid end")
		}
		if to.Before(from) {
			from, to = to, from
		}
		tw = "custom"
	}
	country := firstNonEmpty(q.Get("countryCode"), q.Get("country"))
	return filters{
		Window:      tw,
		From:        from,
		To:          to,
		Q:           strings.TrimSpace(q.Get("q")),
		CountryCode: country,
		Region:      q.Get("region"),
		Sector:      q.Get("sector"),
		Industry:    q.Get("industry"),
		Venue:       q.Get("venue"),
		Symbol:      q.Get("symbol"),
		Locale:      q.Get("locale"),
	}, nil
}

func normalizeReplayViewMode(in string) string {
	switch in {
	case "as_scored", "recomputed":
		return in
	default:
		return "recomputed"
	}
}

func normalizeWindow(in string) string {
	switch in {
	case "1h", "24h", "7d", "30d", "90d", "1y", "5y", "20y", "custom":
		return in
	default:
		return ""
	}
}

func parseFeedLimit(in string, fallback, max int) int {
	if in == "" {
		return fallback
	}
	n, err := strconv.Atoi(in)
	if err != nil || n <= 0 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}

func windowSQLForExecutive(tw string) string {
	switch tw {
	case "1h":
		return "i.last_activity_at > now()-interval '1 hour'"
	case "30d":
		return "i.last_activity_at > now()-interval '30 days'"
	case "90d":
		return "i.last_activity_at > now()-interval '90 days'"
	case "1y":
		return "i.last_activity_at > now()-interval '1 year'"
	case "5y":
		return "i.last_activity_at > now()-interval '5 years'"
	case "20y":
		return "i.last_activity_at > now()-interval '20 years'"
	case "7d":
		return "i.last_activity_at > now()-interval '7 days'"
	default:
		return "i.last_activity_at > now()-interval '24 hours'"
	}
}

func buildExecutiveIndustryWhere(f filters) (string, []any) {
	args := []any{}
	where := windowSQLForExecutive(f.Window)
	if f.Window == "custom" {
		from, to := f.From, f.To
		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			from, to = to, from
		}
		switch {
		case !from.IsZero() && !to.IsZero():
			args = append(args, from, to)
			where = fmt.Sprintf("i.last_activity_at BETWEEN $%d AND $%d", len(args)-1, len(args))
		case !from.IsZero():
			args = append(args, from)
			where = fmt.Sprintf("i.last_activity_at >= $%d", len(args))
		case !to.IsZero():
			args = append(args, to)
			where = fmt.Sprintf("i.last_activity_at <= $%d", len(args))
		}
	}
	return where, args
}
