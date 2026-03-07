package query

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScorePoint is a single data point in the score time series.
type ScorePoint struct {
	Time              time.Time `json:"time"`
	Ts                time.Time `json:"ts"`
	AnomalyScore      float64   `json:"anomaly_score"`
	ScoreNorm         float64   `json:"score_norm"`
	ScoreRaw          float64   `json:"score_raw"`
	EscalationProb    float64   `json:"escalation_prob"`
	Composite         float64   `json:"composite"`
	CompositeRisk     float64   `json:"composite_risk"`
	ModelVersion      string    `json:"model_version"`
	ProducedAt        time.Time `json:"produced_at"`
	ScoringRunID      string    `json:"scoring_run_id"`
	FeatureSnapshot   string    `json:"feature_snapshot_hash"`
	ArtifactHash      string    `json:"artifact_hash"`
	ModelArtifactHash string    `json:"model_artifact_hash"`
	Symbol            string    `json:"symbol"`
	Severity          string    `json:"severity"`
}

// CandlePoint is one OHLCV bucket for the chart.
type CandlePoint struct {
	Ts     time.Time `json:"ts"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
	Symbol string    `json:"symbol"`
}

func windowInterval(window string) string {
	switch window {
	case "1h":
		return "1 hour"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	case "90d":
		return "90 days"
	case "1y":
		return "1 year"
	case "5y":
		return "5 years"
	case "20y":
		return "20 years"
	default:
		return "24 hours"
	}
}

func candleResolution(res string) string {
	switch res {
	case "5m":
		return "5 minutes"
	case "1h":
		return "1 hour"
	case "1d":
		return "1 day"
	default:
		return "1 minute"
	}
}

func scoreSeriesSource(window string) (string, string) {
	switch window {
	case "1y", "5y", "20y":
		return "scores_1d", "bucket"
	case "30d", "90d":
		return "scores_1h", "bucket"
	default:
		return "scores", "ts"
	}
}

func candleSeriesSource(window string) (string, string, bool) {
	switch window {
	case "1y", "5y", "20y":
		return "candles_1d", "bucket", true
	case "30d", "90d":
		return "candles_1h", "bucket", true
	default:
		return "candles", "bucket", false
	}
}

func scoreWindowPredicate(window string, from, to time.Time, startArg int, timeCol string) (string, []any) {
	if window == "custom" && !from.IsZero() && !to.IsZero() {
		return fmt.Sprintf("%s BETWEEN $%d AND $%d", timeCol, startArg, startArg+1), []any{from, to}
	}
	return fmt.Sprintf("%s > now() - interval '%s'", timeCol, windowInterval(window)), nil
}

func scoreSeriesExpressions(source, timeCol string) (string, string) {
	if source == "scores" {
		return fmt.Sprintf(`
				%s AS ts,
				COALESCE(normalized_anomaly_score, score, 0) AS anomaly_score,
				COALESCE(normalized_anomaly_score, score, 0) AS score_norm,
				COALESCE(raw_anomaly_score, score, 0)        AS score_raw,
				COALESCE(escalation_probability, 0)          AS escalation_prob,
				COALESCE(composite_risk, score, 0)           AS composite,
				COALESCE(composite_risk, score, 0)           AS composite_risk,
				COALESCE(model_version, 'baseline-v1')       AS model_version,
				COALESCE(produced_at, ts)                    AS produced_at,
				COALESCE(scoring_run_id::text, '')           AS scoring_run_id,
				COALESCE(feature_snapshot_hash, '')          AS feature_snapshot_hash,
				COALESCE(artifact_hash, '')                  AS artifact_hash,
				COALESCE(model_artifact_hash, '')            AS model_artifact_hash,
				symbol,
				COALESCE(severity, '')                       AS severity,
		`, timeCol), fmt.Sprintf("COALESCE(produced_at, %s)", timeCol)
	}
	return fmt.Sprintf(`
				%s AS ts,
				COALESCE(anomaly_score, 0)             AS anomaly_score,
				COALESCE(score_norm, 0)                AS score_norm,
				COALESCE(score_raw, 0)                 AS score_raw,
				COALESCE(escalation_prob, 0)           AS escalation_prob,
				COALESCE(composite, 0)                 AS composite,
				COALESCE(composite_risk, 0)            AS composite_risk,
				COALESCE(model_version, 'baseline-v1') AS model_version,
				COALESCE(produced_at, %s)              AS produced_at,
				COALESCE(scoring_run_id, '')           AS scoring_run_id,
				COALESCE(feature_snapshot_hash, '')    AS feature_snapshot_hash,
				COALESCE(artifact_hash, '')            AS artifact_hash,
				COALESCE(model_artifact_hash, '')      AS model_artifact_hash,
				symbol,
				COALESCE(severity, '')                 AS severity,
		`, timeCol, timeCol), "COALESCE(produced_at, ts)"
}

func loadScoreSeriesFromSource(ctx context.Context, db *pgxpool.Pool, source, timeCol, symbol, window string, maxPoints int, from, to time.Time) ([]ScorePoint, error) {
	selectCols, orderCol := scoreSeriesExpressions(source, timeCol)
	timeWhere, extraArgs := scoreWindowPredicate(window, from, to, 3, timeCol)

	query := fmt.Sprintf(`
		WITH base AS (
			SELECT
				%s
				row_number() OVER (ORDER BY %s) AS rn,
				count(*) OVER () AS total
			FROM %s
			WHERE %s
			  AND ($1 = '' OR symbol = $1)
		)
		SELECT ts, anomaly_score, score_norm, score_raw, escalation_prob, composite, composite_risk, model_version, produced_at, scoring_run_id, feature_snapshot_hash, artifact_hash, model_artifact_hash, symbol, severity
		FROM base
		WHERE total <= $2 OR rn %% GREATEST(1, total / $2) = 0
		ORDER BY produced_at
		LIMIT $2
	`, selectCols, orderCol, source, timeWhere)

	args := []any{symbol, maxPoints}
	args = append(args, extraArgs...)
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ScorePoint, 0, maxPoints)
	for rows.Next() {
		var p ScorePoint
		if err := rows.Scan(&p.Ts, &p.AnomalyScore, &p.ScoreNorm, &p.ScoreRaw, &p.EscalationProb, &p.Composite, &p.CompositeRisk, &p.ModelVersion, &p.ProducedAt, &p.ScoringRunID, &p.FeatureSnapshot, &p.ArtifactHash, &p.ModelArtifactHash, &p.Symbol, &p.Severity); err != nil {
			continue
		}
		p.Time = p.Ts
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadScoreSeries returns a time-ordered score series for charting.
// symbol="" returns all symbols; window selects the lookback period.
// Results are sampled to at most maxPoints using a simple nth-row approach.
func LoadScoreSeries(ctx context.Context, db *pgxpool.Pool, symbol, window string, maxPoints int, from, to time.Time) ([]ScorePoint, error) {
	if maxPoints <= 0 {
		maxPoints = 500
	}
	source, timeCol := scoreSeriesSource(window)
	out, err := loadScoreSeriesFromSource(ctx, db, source, timeCol, symbol, window, maxPoints, from, to)
	if (err != nil || len(out) == 0) && source != "scores" {
		return loadScoreSeriesFromSource(ctx, db, "scores", "ts", symbol, window, maxPoints, from, to)
	}
	return out, err
}

func loadCandleSeriesFromSource(ctx context.Context, db *pgxpool.Pool, source, timeCol string, aggregated bool, symbol, window, res string, maxPoints int) ([]CandlePoint, error) {
	interval := windowInterval(window)
	resolution := candleResolution(res)

	var query string
	if aggregated {
		query = fmt.Sprintf(`
			WITH joined AS (
				SELECT
					%s AS ts,
					open,
					high,
					low,
					close,
					volume,
					symbol,
					row_number() OVER (ORDER BY %s) AS rn,
					count(*) OVER () AS total
				FROM %s
				WHERE %s > now() - interval '%s'
				  AND ($1 = '' OR symbol = $1)
			)
			SELECT ts, open, high, low, close, volume, symbol
			FROM joined
			WHERE total <= $2 OR rn %% GREATEST(1, total / $2) = 0
			ORDER BY ts
			LIMIT $2
		`, timeCol, timeCol, source, timeCol, interval)
	} else {
		query = fmt.Sprintf(`
			WITH seeded AS (
				SELECT
					bucket,
					open,
					high,
					low,
					close,
					volume,
					symbol
				FROM %s
				WHERE %s > now() - interval '%s'
				  AND ($1 = '' OR symbol = $1)
				ORDER BY %s DESC
				LIMIT GREATEST($2 * 24, 600)
			),
			filtered AS (
				SELECT
					time_bucket('%s', bucket) AS grp,
					bucket,
					open,
					high,
					low,
					close,
					volume,
					symbol
				FROM seeded
			),
			ranked AS (
				SELECT
					grp,
					symbol,
					open,
					high,
					low,
					close,
					volume,
					row_number() OVER (PARTITION BY grp, symbol ORDER BY bucket ASC) AS rn_open,
					row_number() OVER (PARTITION BY grp, symbol ORDER BY bucket DESC) AS rn_close
				FROM filtered
			),
			agg AS (
				SELECT
					grp AS ts,
					symbol,
					MAX(CASE WHEN rn_open = 1 THEN open END) AS open,
					MAX(high) AS high,
					MIN(low) AS low,
					MAX(CASE WHEN rn_close = 1 THEN close END) AS close,
					SUM(volume) AS volume
				FROM ranked
				GROUP BY grp, symbol
			),
			joined AS (
				SELECT
					a.ts,
					a.open,
					a.high,
					a.low,
					a.close,
					a.volume,
					a.symbol,
					row_number() OVER (ORDER BY a.ts) AS rn,
					count(*) OVER () AS total
				FROM agg a
			)
			SELECT ts, open, high, low, close, volume, symbol
			FROM joined
			WHERE total <= $2 OR rn %% GREATEST(1, total / $2) = 0
			ORDER BY ts
			LIMIT $2
		`, source, timeCol, interval, timeCol, resolution)
	}

	rows, err := db.Query(ctx, query, symbol, maxPoints)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CandlePoint, 0, maxPoints)
	for rows.Next() {
		var p CandlePoint
		if err := rows.Scan(&p.Ts, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Symbol); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadCandleSeries returns OHLCV candles for the charting panel.
func LoadCandleSeries(ctx context.Context, db *pgxpool.Pool, symbol, window, res string, maxPoints int) ([]CandlePoint, error) {
	if maxPoints <= 0 {
		maxPoints = 500
	}
	source, timeCol, aggregated := candleSeriesSource(window)
	out, err := loadCandleSeriesFromSource(ctx, db, source, timeCol, aggregated, symbol, window, res, maxPoints)
	if (err != nil || len(out) == 0) && source != "candles" {
		return loadCandleSeriesFromSource(ctx, db, "candles", "bucket", false, symbol, window, res, maxPoints)
	}
	return out, err
}
