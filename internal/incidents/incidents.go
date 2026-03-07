package incidents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultClusterWindow = 60 * time.Second

func SeverityBandForScore(compositeRisk float64) string {
	if compositeRisk >= 0.85 {
		return "critical"
	}
	if compositeRisk >= 0.7 {
		return "high"
	}
	if compositeRisk >= 0.45 {
		return "elevated"
	}
	return "stable"
}

func FindOrCreateOpenIncident(ctx context.Context, pool *pgxpool.Pool, symbol string, score, escalation, confidence float64, featureHash, modelVersion string, at time.Time) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM incidents WHERE primary_symbol=$1 AND status IN ('open','ack') AND last_activity_at >= $2 ORDER BY id DESC LIMIT 1`, symbol, at.Add(-DefaultClusterWindow)).Scan(&id)
	rec := RecommendedActionForIncident(score, confidence, "stable", 0, false)
	reason := RankReasonForIncident(score, escalation, confidence, "stable", 0, false)
	severity := SeverityBandForScore(score)
	if err == nil {
		_, err = pool.Exec(ctx, `UPDATE incidents SET last_activity_at=$2,updated_at=now(),severity_band=$3,priority_score=GREATEST(priority_score,$4),composite_risk=GREATEST(composite_risk,$4),escalation_probability=GREATEST(escalation_probability,$5),confidence=LEAST(confidence,$6),top_driver_1=$7,top_driver_2=$8 WHERE id=$1`, id, at, severity, score, escalation, confidence, rec, reason)
		return id, err
	}
	err = pool.QueryRow(ctx, `INSERT INTO incidents (primary_symbol,status,severity_band,priority_score,composite_risk,escalation_probability,confidence,trust_state,feature_snapshot_hash,feature_set_version,model_version,started_at,last_activity_at,created_at,updated_at,top_driver_1,top_driver_2) VALUES ($1,'open',$2,$3,$3,$4,$5,'stable',$6,'v1',$7,$8,$8,now(),now(),$9,$10) RETURNING id`, symbol, severity, score, escalation, confidence, featureHash, modelVersion, at, rec, reason).Scan(&id)
	return id, err
}

func CanPromoteToCase(status string) bool {
	return status == "open" || status == "ack"
}

func NextIncidentStatus(current, command string) (string, error) {
	switch command {
	case "ack":
		if current == "open" {
			return "ack", nil
		}
	case "escalate":
		if current == "open" || current == "ack" {
			return "ack", nil
		}
	case "resolve", "close":
		if current == "open" || current == "ack" || current == "suppressed" {
			return "resolved", nil
		}
	case "suppress":
		if current == "open" || current == "ack" {
			return "suppressed", nil
		}
	}
	return "", fmt.Errorf("invalid transition %s -> %s", current, command)
}

func RecommendedActionForIncident(compositeRisk, confidence float64, trustState string, recurrence float64, suppressed bool) string {
	if suppressed {
		return "suppressed_by_policy"
	}
	if trustState != "stable" || confidence < 0.45 {
		return "check_data_quality"
	}
	if compositeRisk >= 0.8 && recurrence >= 0.3 {
		return "promote_to_case"
	}
	if compositeRisk >= 0.75 {
		return "review_now"
	}
	return "watch"
}

func RankReasonForIncident(compositeRisk, escalation, confidence float64, trustState string, recurrence float64, openSLA bool) string {
	if trustState != "stable" {
		return "Low trust reduced priority"
	}
	if recurrence >= 0.6 && openSLA {
		return "Recurring incident with open SLA"
	}
	if compositeRisk >= 0.75 && escalation >= 0.6 {
		return "High anomaly + elevated escalation"
	}
	if confidence < 0.5 {
		return "Low confidence requires validation"
	}
	return "Elevated risk under watch"
}

func SafetyLevelForIncident(compositeRisk, dqPenalty float64) string {
	if dqPenalty >= 0.4 {
		return "Data Unreliable"
	}
	if compositeRisk >= 0.85 {
		return "Critical"
	}
	if compositeRisk >= 0.7 {
		return "High Risk"
	}
	if compositeRisk >= 0.45 {
		return "Elevated"
	}
	return "Stable"
}

func TrustLabelForIncident(trustState string, confidence float64) string {
	if strings.ToLower(trustState) != "stable" || confidence < 0.5 {
		return "degraded"
	}
	return "stable"
}
