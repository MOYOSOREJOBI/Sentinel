package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadTrust(ctx context.Context, db *pgxpool.Pool, f QueueFilters) (map[string]any, error) {
	where, args := whereClause(f)
	featureWhere, featureArgs := featureWhereClause(f)
	meta := readModelBundleMetadata()

	var dq, fallback, total int
	_ = db.QueryRow(ctx, `SELECT count(s.id) FROM scores s LEFT JOIN incidents i ON i.primary_symbol=s.symbol LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE coalesce(s.priority_score,0)<1 AND `+where, args...).Scan(&dq)
	_ = db.QueryRow(ctx, `SELECT count(s.id) FROM scores s LEFT JOIN incidents i ON i.primary_symbol=s.symbol LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE `+where, args...).Scan(&total)
	_ = db.QueryRow(ctx, `SELECT count(s.id) FROM scores s LEFT JOIN incidents i ON i.primary_symbol=s.symbol LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE coalesce(s.model_version,'') LIKE '%fallback%' AND `+where, args...).Scan(&fallback)

	var suppressedAlerts int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM alerts WHERE status='suppressed'`).Scan(&suppressedAlerts)
	var circuitBreakers int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM circuit_suppression WHERE suppressed=true`).Scan(&circuitBreakers)
	var missingness, duplicates, lateEvents float64
	_ = db.QueryRow(ctx, `SELECT
COALESCE(avg(COALESCE((ft.payload->'payload'->>'missingness_ratio')::double precision, 0)), 0),
COALESCE(avg(COALESCE((ft.payload->'payload'->>'duplicate_ratio')::double precision, 0)), 0),
COALESCE(avg(COALESCE((ft.payload->'payload'->>'late_ratio')::double precision, 0)), 0)
FROM features ft
LEFT JOIN instrument_metadata m ON m.instrument_id=ft.symbol
WHERE `+featureWhere, featureArgs...).Scan(&missingness, &duplicates, &lateEvents)

	var latestModelVersion, latestModelArtifactHash, latestScoringRunID string
	var latestProducedAt any
	_ = db.QueryRow(ctx, `SELECT coalesce(s.model_version,''), coalesce(s.model_artifact_hash,''), coalesce(s.scoring_run_id::text,''), coalesce(s.produced_at, s.ts) FROM scores s LEFT JOIN incidents i ON i.primary_symbol=s.symbol LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE `+where+` ORDER BY coalesce(s.produced_at, s.ts) DESC LIMIT 1`, args...).Scan(&latestModelVersion, &latestModelArtifactHash, &latestScoringRunID, &latestProducedAt)
	var activeModelVersion, activeArtifactHash string
	var activeDeployedAt any
	_ = db.QueryRow(ctx, `SELECT model_version, artifact_hash, coalesce(deployed_at, created_at) FROM model_deployments WHERE status='deployed' ORDER BY coalesce(deployed_at, created_at) DESC NULLS LAST, created_at DESC LIMIT 1`).Scan(&activeModelVersion, &activeArtifactHash, &activeDeployedAt)

	fallbackMode := meta.FallbackMode
	if latestModelVersion != "" {
		fallbackMode = strings.Contains(strings.ToLower(latestModelVersion), "fallback")
	}
	modelVersion := firstNonEmptyString(activeModelVersion, latestModelVersion)
	if modelVersion == "" {
		modelVersion = meta.ModelVersion
	}
	modelArtifactHash := firstNonEmptyString(activeArtifactHash, latestModelArtifactHash)
	if modelArtifactHash == "" {
		modelArtifactHash = meta.ArtifactHash
	}

	cacheHealth := "ok"
	if total == 0 {
		cacheHealth = "warming"
	}
	state := "stable"
	if fallbackMode {
		state = "degraded"
	}
	return map[string]any{
		"dqTotals": map[string]any{"warnings": dq},
		"dqBreakdown": map[string]any{
			"missingness": map[string]any{"value": missingness, "threshold": 0.02},
			"duplicates":  map[string]any{"value": duplicates, "threshold": 0.01},
			"lateEvents":  map[string]any{"value": lateEvents, "threshold": 0.005},
		},
		"modelDegradedCount":   fallback,
		"cacheHealth":          cacheHealth,
		"circuitBreakerCount":  circuitBreakers,
		"suppressedAlertCount": suppressedAlerts,
		"trustSummary":         map[string]any{"state": state, "fallbackRatio": ratioInt(fallback, total)},
		"modelState": map[string]any{
			"fallback_mode":         fallbackMode,
			"model_version":         modelVersion,
			"active_model_version":  modelVersion,
			"last_trained_at":       meta.TrainedAt,
			"model_artifact_hash":   modelArtifactHash,
			"active_artifact_hash":  modelArtifactHash,
			"active_deployed_at":    activeDeployedAt,
			"latest_scoring_run_id": latestScoringRunID,
			"latest_produced_at":    latestProducedAt,
			"calibration_version":   meta.CalibrationVersion,
		},
		"calibration": map[string]any{
			"auc":               meta.Metrics.AUC,
			"brier":             meta.Metrics.Brier,
			"pr_auc":            meta.Metrics.PRAUC,
			"anomaly_stability": meta.Metrics.AnomalyStability,
			"history": []map[string]any{
				{
					"trained_at":        meta.TrainedAt,
					"auc":               meta.Metrics.AUC,
					"brier":             meta.Metrics.Brier,
					"pr_auc":            meta.Metrics.PRAUC,
					"anomaly_stability": meta.Metrics.AnomalyStability,
				},
			},
		},
	}, nil
}

func featureWhereClause(f QueueFilters) (string, []any) {
	args := []any{}
	where := strings.ReplaceAll(windowSQL(f.Window), "i.last_activity_at", "ft.ts")
	if f.Window == "custom" {
		from, to := f.From, f.To
		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			from, to = to, from
		}
		switch {
		case !from.IsZero() && !to.IsZero():
			args = append(args, from, to)
			where = fmt.Sprintf("ft.ts BETWEEN $%d AND $%d", len(args)-1, len(args))
		case !from.IsZero():
			args = append(args, from)
			where = fmt.Sprintf("ft.ts >= $%d", len(args))
		case !to.IsZero():
			args = append(args, to)
			where = fmt.Sprintf("ft.ts <= $%d", len(args))
		}
	}
	add := func(col, val string) {
		if strings.TrimSpace(val) == "" {
			return
		}
		args = append(args, val)
		where += fmt.Sprintf(" AND %s=$%d", col, len(args))
	}
	add("m.region", f.Region)
	add("coalesce(m.country_code,'XX')", f.CountryCode)
	add("m.sector", f.Sector)
	add("m.industry", f.Industry)
	add("m.venue", f.Venue)
	add("ft.symbol", f.Symbol)
	if q := strings.TrimSpace(strings.ToLower(f.Q)); q != "" {
		args = append(args, "%"+q+"%")
		idx := len(args)
		where += fmt.Sprintf(" AND (lower(ft.symbol) LIKE $%d OR lower(coalesce(m.country_code,'')) LIKE $%d OR lower(coalesce(m.region,'')) LIKE $%d OR lower(coalesce(m.sector,'')) LIKE $%d OR lower(coalesce(m.industry,'')) LIKE $%d OR lower(coalesce(m.venue,'')) LIKE $%d)", idx, idx, idx, idx, idx, idx)
	}
	return where, args
}

func ratioInt(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
