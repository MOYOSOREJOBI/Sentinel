package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type scorePoint struct {
	Key         string
	Symbol      string
	Timestamp   time.Time
	Score       float64
	Severity    string
	Explanation string
}

func buildReplayDiffSummary(ctx context.Context, db *pgxpool.Pool, job Job, result Result) (DiffSummary, error) {
	asScoredRows, err := loadScorePoints(ctx, db, `SELECT symbol, ts, score, severity, explanation, ordinal
FROM (
	SELECT symbol, ts, score, severity, coalesce(explanation,'') AS explanation,
		row_number() OVER (PARTITION BY symbol, ts ORDER BY idempotency_key) AS ordinal
	FROM scores
	WHERE ts BETWEEN $1 AND $2 AND coalesce(replay_run_id,'')=''
) ranked
ORDER BY ts, symbol, ordinal`, job.TimeWindowStart, job.TimeWindowEnd)
	if err != nil {
		return DiffSummary{}, err
	}
	recomputedRows, err := loadScorePoints(ctx, db, `SELECT symbol, ts, score, severity, explanation, ordinal
FROM (
	SELECT symbol, ts, score, severity, coalesce(explanation,'') AS explanation,
		row_number() OVER (PARTITION BY symbol, ts ORDER BY idempotency_key) AS ordinal
	FROM scores
	WHERE replay_run_id=$1
) ranked
ORDER BY ts, symbol, ordinal`, job.ID)
	if err != nil {
		return DiffSummary{}, err
	}
	prevReplayID, prevFingerprint := loadPreviousReplayFingerprint(ctx, db, job)
	return summarizeReplayDiff(result, asScoredRows, recomputedRows, prevReplayID, prevFingerprint), nil
}

func loadScorePoints(ctx context.Context, db *pgxpool.Pool, query string, args ...any) ([]scorePoint, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []scorePoint{}
	for rows.Next() {
		var p scorePoint
		var ordinal int
		if err := rows.Scan(&p.Symbol, &p.Timestamp, &p.Score, &p.Severity, &p.Explanation, &ordinal); err != nil {
			return nil, err
		}
		p.Key = fmt.Sprintf("%s|%s|%06d", p.Symbol, p.Timestamp.UTC().Format(time.RFC3339Nano), ordinal)
		out = append(out, p)
	}
	return out, rows.Err()
}

func loadPreviousReplayFingerprint(ctx context.Context, db *pgxpool.Pool, job Job) (string, string) {
	var previousID string
	var raw []byte
	err := db.QueryRow(ctx, `SELECT j.id::text, rr.diff_summary
FROM replay_jobs j
JOIN replay_runs rr ON rr.id = j.id::text
WHERE j.id::text <> $1
  AND j.status = 'completed'
  AND j.time_window_start = $2
  AND j.time_window_end = $3
  AND j.replay_mode = $4
  AND j.model_version = $5
  AND j.feature_set_version = $6
ORDER BY j.completed_at DESC NULLS LAST, j.requested_at DESC
LIMIT 1`, job.ID, job.TimeWindowStart, job.TimeWindowEnd, job.ReplayMode, job.ModelVersion, job.FeatureSetVersion).Scan(&previousID, &raw)
	if err != nil {
		return "", ""
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		return previousID, ""
	}
	fingerprint, _ := summary["recomputed_fingerprint"].(string)
	return previousID, fingerprint
}

func summarizeReplayDiff(result Result, asScoredRows, recomputedRows []scorePoint, previousReplayID, previousFingerprint string) DiffSummary {
	asStats := summarizeScoreStats(asScoredRows)
	recStats := summarizeScoreStats(recomputedRows)
	asFingerprint := fingerprintScoreRows(asScoredRows)
	recFingerprint := fingerprintScoreRows(recomputedRows)

	type pair struct {
		as  *scorePoint
		rec *scorePoint
	}
	joined := map[string]pair{}
	for i := range asScoredRows {
		item := joined[asScoredRows[i].Key]
		item.as = &asScoredRows[i]
		joined[asScoredRows[i].Key] = item
	}
	for i := range recomputedRows {
		item := joined[recomputedRows[i].Key]
		item.rec = &recomputedRows[i]
		joined[recomputedRows[i].Key] = item
	}

	keys := make([]string, 0, len(joined))
	for key := range joined {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	matched := 0
	mismatched := 0
	missingAsScored := 0
	missingRecomputed := 0
	maxScoreDelta := 0.0
	causeCounts := map[string]int{}
	samples := make([]MismatchSample, 0, 5)

	for _, key := range keys {
		entry := joined[key]
		switch {
		case entry.as == nil:
			missingAsScored++
			causeCounts["missing_as_scored"]++
			if len(samples) < 5 && entry.rec != nil {
				samples = append(samples, MismatchSample{
					Key:                key,
					Symbol:             entry.rec.Symbol,
					Timestamp:          entry.rec.Timestamp.UTC().Format(time.RFC3339),
					RecomputedScore:    entry.rec.Score,
					RecomputedSeverity: entry.rec.Severity,
					Cause:              "missing_as_scored",
				})
			}
		case entry.rec == nil:
			missingRecomputed++
			causeCounts["missing_recomputed"]++
			if len(samples) < 5 {
				samples = append(samples, MismatchSample{
					Key:              key,
					Symbol:           entry.as.Symbol,
					Timestamp:        entry.as.Timestamp.UTC().Format(time.RFC3339),
					AsScoredScore:    entry.as.Score,
					AsScoredSeverity: entry.as.Severity,
					Cause:            "missing_recomputed",
				})
			}
		default:
			delta := math.Abs(entry.rec.Score - entry.as.Score)
			scoreMismatch := delta > 1e-9
			severityMismatch := entry.rec.Severity != entry.as.Severity
			if !scoreMismatch && !severityMismatch {
				matched++
				continue
			}
			mismatched++
			if delta > maxScoreDelta {
				maxScoreDelta = delta
			}
			cause := classifyMismatch(scoreMismatch, severityMismatch)
			causeCounts[cause]++
			if len(samples) < 5 {
				samples = append(samples, MismatchSample{
					Key:                key,
					Symbol:             entry.rec.Symbol,
					Timestamp:          entry.rec.Timestamp.UTC().Format(time.RFC3339),
					AsScoredScore:      entry.as.Score,
					RecomputedScore:    entry.rec.Score,
					ScoreDelta:         entry.rec.Score - entry.as.Score,
					AsScoredSeverity:   entry.as.Severity,
					RecomputedSeverity: entry.rec.Severity,
					Cause:              cause,
				})
			}
		}
	}

	parityStatus := "MATCH"
	parityExplanation := fmt.Sprintf("Replay matched %d stored score points for the selected window.", matched)
	if mismatched > 0 || missingAsScored > 0 || missingRecomputed > 0 {
		parityStatus = "MISMATCH"
		parityExplanation = fmt.Sprintf("Replay diverged: %d mismatched, %d missing from stored scoring, %d missing from replay.", mismatched, missingAsScored, missingRecomputed)
	}
	if len(asScoredRows) == 0 && len(recomputedRows) == 0 {
		parityExplanation = "No score points were present in the selected window."
	}

	determinismStatus := "FIRST_RUN"
	determinismExplanation := "No prior replay with the same window and model inputs was found; this fingerprint is the baseline for future comparisons."
	if previousFingerprint != "" {
		if previousFingerprint == recFingerprint {
			determinismStatus = "MATCH"
			determinismExplanation = fmt.Sprintf("Replay fingerprint matched prior replay %s for the same window and inputs.", previousReplayID)
		} else {
			determinismStatus = "MISMATCH"
			determinismExplanation = fmt.Sprintf("Replay fingerprint differed from prior replay %s for the same window and inputs.", previousReplayID)
		}
	}

	return DiffSummary{
		Result:                 result,
		AsScored:               asStats,
		Recomputed:             recStats,
		AvgScoreDelta:          recStats.AvgScore - asStats.AvgScore,
		HighCountDelta:         recStats.HighOrHigher - asStats.HighOrHigher,
		MatchedCount:           matched,
		MismatchedCount:        mismatched,
		MissingAsScoredCount:   missingAsScored,
		MissingRecomputedCount: missingRecomputed,
		MaxScoreDelta:          maxScoreDelta,
		ParityStatus:           parityStatus,
		ParityExplanation:      parityExplanation,
		CommonMismatchCauses:   topMismatchCauses(causeCounts),
		MismatchSamples:        samples,
		AsScoredFingerprint:    asFingerprint,
		RecomputedFingerprint:  recFingerprint,
		DeterminismStatus:      determinismStatus,
		DeterminismExplanation: determinismExplanation,
		PreviousReplayID:       previousReplayID,
	}
}

func summarizeScoreStats(rows []scorePoint) ScoreStats {
	if len(rows) == 0 {
		return ScoreStats{}
	}
	total := 0.0
	highOrHigher := 0
	for _, row := range rows {
		total += row.Score
		if row.Severity == "high" || row.Severity == "critical" {
			highOrHigher++
		}
	}
	return ScoreStats{
		Count:        len(rows),
		AvgScore:     total / float64(len(rows)),
		HighOrHigher: highOrHigher,
	}
}

func fingerprintScoreRows(rows []scorePoint) string {
	h := sha256.New()
	for _, row := range rows {
		_, _ = fmt.Fprintf(h, "%s|%.12f|%s|%s\n", row.Key, row.Score, row.Severity, row.Explanation)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func classifyMismatch(scoreMismatch, severityMismatch bool) string {
	switch {
	case scoreMismatch && severityMismatch:
		return "score_and_severity_delta"
	case scoreMismatch:
		return "score_delta"
	case severityMismatch:
		return "severity_delta"
	default:
		return "matched"
	}
}

func topMismatchCauses(counts map[string]int) []string {
	type causeCount struct {
		cause string
		count int
	}
	out := make([]causeCount, 0, len(counts))
	for cause, count := range counts {
		out = append(out, causeCount{cause: cause, count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count == out[j].count {
			return out[i].cause < out[j].cause
		}
		return out[i].count > out[j].count
	})
	limit := len(out)
	if limit > 3 {
		limit = 3
	}
	values := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		values = append(values, fmt.Sprintf("%s:%d", strings.ReplaceAll(out[i].cause, "_", " "), out[i].count))
	}
	return values
}
