package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sentinel/internal/metrics"
	"sentinel/internal/pipeline/scoring"
)

func Run(ctx context.Context, db *pgxpool.Pool, jobID string) error {
	if err := markReplayJobRunning(ctx, db, jobID); err != nil {
		return err
	}
	metrics.IncReplayJobEvent("governance", "running")
	job, err := loadReplayJob(ctx, db, jobID)
	if err != nil {
		_ = markReplayJobFailed(ctx, db, jobID, err.Error())
		metrics.IncReplayJobEvent("governance", "failed")
		return err
	}
	ticks, err := loadRawTicks(ctx, db, job.TimeWindowStart, job.TimeWindowEnd)
	if err != nil {
		_ = markReplayJobFailed(ctx, db, jobID, err.Error())
		metrics.IncReplayJobEvent("governance", "failed")
		return err
	}
	sortTicks(ticks)
	result := Result{}
	if job.ReplayMode != "as_scored" {
		result, err = recomputePipeline(ctx, db, job, ticks)
		if err != nil {
			_ = markReplayJobFailed(ctx, db, jobID, err.Error())
			metrics.IncReplayJobEvent("governance", "failed")
			return err
		}
	}
	if err := persistReplayResult(ctx, db, job, result); err != nil {
		_ = markReplayJobFailed(ctx, db, jobID, err.Error())
		metrics.IncReplayJobEvent("governance", "failed")
		return err
	}
	if err := markReplayJobCompleted(ctx, db, jobID); err != nil {
		metrics.IncReplayJobEvent("governance", "failed")
		return err
	}
	metrics.IncReplayJobEvent("governance", "completed")
	return nil
}

func sortTicks(ticks []Tick) {
	sort.Slice(ticks, func(i, j int) bool {
		if ticks[i].EventTime.Equal(ticks[j].EventTime) {
			return ticks[i].SequenceID < ticks[j].SequenceID
		}
		return ticks[i].EventTime.Before(ticks[j].EventTime)
	})
}

func recomputePipeline(ctx context.Context, db *pgxpool.Pool, job Job, ticks []Tick) (Result, error) {
	if len(ticks) == 0 {
		return Result{}, nil
	}
	feat, score, candle := 0, 0, 0
	lastPrice := map[string]float64{}
	for _, t := range ticks {
		payload := map[string]any{"symbol": t.Symbol, "price": t.Price, "volume": t.Volume, "event_time": t.EventTime, "replay_job_id": job.ID}
		b, _ := json.Marshal(payload)
		bucket := t.EventTime.Truncate(time.Second)
		_, _ = db.Exec(ctx, `INSERT INTO candles(idempotency_key,symbol,bucket,interval,open,high,low,close,volume,replay_run_id) VALUES($1,$2,$3,'1s',$4,$4,$4,$4,$5,$6) ON CONFLICT DO NOTHING`, fmt.Sprintf("replay:candle:%s:%s", job.ID, t.EventID), t.Symbol, bucket, t.Price, t.Volume, job.ID)
		candle++
		_, _ = db.Exec(ctx, `INSERT INTO features(idempotency_key,symbol,ts,feature_hash,payload,replay_run_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, fmt.Sprintf("replay:feature:%s:%s", job.ID, t.EventID), t.Symbol, t.EventTime, job.ID, b, job.ID)
		feat++

		s, sev := scoring.ScoreAndSeverity(lastPrice[t.Symbol], t.Price)
		explanation := fmt.Sprintf("recomputed_return=%.6f", s)
		_, _ = db.Exec(ctx, `INSERT INTO scores(idempotency_key,symbol,ts,score,severity,explanation,replay_run_id,model_version,feature_set_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`, fmt.Sprintf("replay:score:%s:%s", job.ID, t.EventID), t.Symbol, t.EventTime, s, sev, explanation, job.ID, job.ModelVersion, job.FeatureSetVersion)
		lastPrice[t.Symbol] = t.Price
		score++
	}
	return Result{TickCount: len(ticks), CandleCount: candle, FeatureCount: feat, ScoreCount: score}, nil
}

func persistReplayResult(ctx context.Context, db *pgxpool.Pool, job Job, result Result) error {
	diff, err := buildReplayDiffSummary(ctx, db, job, result)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(diff)
	_, err = db.Exec(ctx, `INSERT INTO replay_runs(id,incident_id,status,diff_summary,completed_at) VALUES($1,$2,'completed',$3,now()) ON CONFLICT (id) DO UPDATE SET status='completed',diff_summary=$3,completed_at=now()`, job.ID, job.ReplayMode, b)
	_, err = db.Exec(ctx, `INSERT INTO replay_runs(id,incident_id,status,diff_summary,completed_at) VALUES($1,'recompute','completed',$2,now()) ON CONFLICT (id) DO UPDATE SET status='completed',diff_summary=$2,completed_at=now()`, job.ID, b)
	return err
}

func buildReplayDiffSummary(ctx context.Context, db *pgxpool.Pool, job Job, result Result) (DiffSummary, error) {
	as := ScoreStats{}
	rec := ScoreStats{}
	if err := db.QueryRow(ctx, `SELECT count(*),coalesce(avg(score),0),count(*) FILTER (WHERE severity in ('high','critical')) FROM scores WHERE ts BETWEEN $1 AND $2 AND coalesce(replay_run_id,'')=''`, job.TimeWindowStart, job.TimeWindowEnd).Scan(&as.Count, &as.AvgScore, &as.HighOrHigher); err != nil {
		return DiffSummary{}, err
	}
	if err := db.QueryRow(ctx, `SELECT count(*),coalesce(avg(score),0),count(*) FILTER (WHERE severity in ('high','critical')) FROM scores WHERE replay_run_id=$1`, job.ID).Scan(&rec.Count, &rec.AvgScore, &rec.HighOrHigher); err != nil {
		return DiffSummary{}, err
	}
	return DiffSummary{
		Result:         result,
		AsScored:       as,
		Recomputed:     rec,
		AvgScoreDelta:  rec.AvgScore - as.AvgScore,
		HighCountDelta: rec.HighOrHigher - as.HighOrHigher,
	}, nil
}
