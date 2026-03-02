package replay

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func markReplayJobRunning(ctx context.Context, db *pgxpool.Pool, jobID string) error {
	_, err := db.Exec(ctx, `UPDATE replay_jobs SET status='running',started_at=now(),error_message=NULL WHERE id=$1`, jobID)
	return err
}
func markReplayJobFailed(ctx context.Context, db *pgxpool.Pool, jobID, msg string) error {
	_, err := db.Exec(ctx, `UPDATE replay_jobs SET status='failed',completed_at=now(),error_message=$2 WHERE id=$1`, jobID, msg)
	return err
}
func markReplayJobCompleted(ctx context.Context, db *pgxpool.Pool, jobID string) error {
	_, err := db.Exec(ctx, `UPDATE replay_jobs SET status='completed',completed_at=now() WHERE id=$1`, jobID)
	return err
}

func loadReplayJob(ctx context.Context, db *pgxpool.Pool, jobID string) (Job, error) {
	var j Job
	err := db.QueryRow(ctx, `SELECT id::text,status,requested_by,time_window_start,time_window_end,watermark_policy_id,allowed_lateness_ms,model_version,feature_set_version,replay_mode FROM replay_jobs WHERE id=$1`, jobID).Scan(&j.ID, &j.Status, &j.RequestedBy, &j.TimeWindowStart, &j.TimeWindowEnd, &j.WatermarkPolicyID, &j.AllowedLatenessMS, &j.ModelVersion, &j.FeatureSetVersion, &j.ReplayMode)
	return j, err
}

func loadRawTicks(ctx context.Context, db *pgxpool.Pool, s, e time.Time) ([]Tick, error) {
	rows, err := db.Query(ctx, `SELECT event_id,symbol,price,volume,event_time,0 FROM raw_ticks WHERE event_time BETWEEN $1 AND $2 ORDER BY event_time ASC`, s, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tick{}
	for rows.Next() {
		var t Tick
		if rows.Scan(&t.EventID, &t.Symbol, &t.Price, &t.Volume, &t.EventTime, &t.SequenceID) == nil {
			out = append(out, t)
		}
	}
	return out, nil
}
