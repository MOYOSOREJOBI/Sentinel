package query

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type feedRow struct {
	ID                    int64
	Symbol                string
	Status                string
	SeverityBand          string
	PriorityScore         float64
	CompositeRisk         float64
	EscalationProbability float64
	Confidence            float64
	TrustState            string
	ModelVersion          string
	TopDriver1            string
	TopDriver2            string
	LastActivityAt        time.Time
	CountryCode           string
	Region                string
	Industry              string
	Sector                string
	Venue                 string
}

type feedRanked struct {
	row               feedRow
	rankScore         float64
	marketImpliedRisk float64
	components        map[string]float64
	rankReason        []string
}

type FeedHistory struct {
	AvailableDays int       `json:"available_days"`
	OldestAt      time.Time `json:"oldest_at"`
	NewestAt      time.Time `json:"newest_at"`
}

type FeedPage struct {
	Items          []map[string]any `json:"items"`
	NextCursor     string           `json:"next_cursor"`
	ServerTime     time.Time        `json:"server_time"`
	History        FeedHistory      `json:"history"`
	HistoryLimited bool             `json:"history_limited"`
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func severityWeight(sev string) float64 {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return 1.0
	case "high":
		return 0.8
	case "elevated":
		return 0.6
	default:
		return 0.45
	}
}

func activeFilterCount(f QueueFilters) int {
	n := 0
	for _, v := range []string{f.Q, f.CountryCode, f.Region, f.Sector, f.Industry, f.Venue, f.Symbol} {
		if strings.TrimSpace(v) != "" {
			n++
		}
	}
	return n
}

func rankReasonList(row feedRow, confidenceNorm, freshnessNorm, trustPenalty float64) []string {
	reasons := []string{
		fmt.Sprintf("severity:%s", strings.ToLower(strings.TrimSpace(row.SeverityBand))),
		fmt.Sprintf("priority:%.2f", clamp01(row.PriorityScore/100.0)),
		fmt.Sprintf("composite:%.2f", clamp01(row.CompositeRisk/100.0)),
		fmt.Sprintf("escalation:%.2f", clamp01(row.EscalationProbability)),
		fmt.Sprintf("confidence:%.2f", confidenceNorm),
		fmt.Sprintf("freshness:%.2f", freshnessNorm),
	}
	if strings.TrimSpace(row.TopDriver1) != "" {
		reasons = append(reasons, "driver:"+row.TopDriver1)
	}
	if trustPenalty > 0 {
		reasons = append(reasons, fmt.Sprintf("trust_penalty:%.2f", trustPenalty))
	}
	return reasons
}

// rankFeedRow computes a bounded [0,1] rank score for a feed item.
// mode: "" or "default" = standard weights; "trending" = boost freshness;
// "for_you" = double the filter-context boost.
func rankFeedRow(row feedRow, f QueueFilters, now time.Time, mode string) (float64, float64, map[string]float64, []string) {
	priorityNorm := clamp01(row.PriorityScore / 100.0)
	compositeNorm := clamp01(row.CompositeRisk / 100.0)
	escalationNorm := clamp01(row.EscalationProbability)
	confidenceNorm := clamp01(row.Confidence)
	severityNorm := severityWeight(row.SeverityBand)

	ageMin := now.Sub(row.LastActivityAt).Minutes()
	if ageMin < 0 {
		ageMin = 0
	}
	freshnessNorm := math.Exp(-ageMin / 180.0)

	trustPenalty := 0.0
	trustState := strings.ToLower(strings.TrimSpace(row.TrustState))
	if trustState != "" && trustState != "stable" && trustState != "healthy" {
		trustPenalty += 0.08
	}
	if strings.Contains(strings.ToLower(row.ModelVersion), "fallback") {
		trustPenalty += 0.10
	}

	filterBoostMax := 0.15
	if mode == "for_you" {
		filterBoostMax = 0.30
	}
	filterBoost := math.Min(filterBoostMax, float64(activeFilterCount(f))*0.025)

	wPriority := 0.28
	wFreshness := 0.12
	if mode == "trending" {
		wPriority = 0.20
		wFreshness = 0.20
	}

	score := wPriority*priorityNorm +
		0.22*compositeNorm +
		0.18*escalationNorm +
		0.12*confidenceNorm +
		wFreshness*freshnessNorm +
		0.08*severityNorm -
		trustPenalty +
		filterBoost
	score = clamp01(score)

	marketImpliedRisk := clamp01(0.60*escalationNorm + 0.40*compositeNorm)
	components := map[string]float64{
		"priority":      priorityNorm,
		"composite":     compositeNorm,
		"escalation":    escalationNorm,
		"confidence":    confidenceNorm,
		"freshness":     freshnessNorm,
		"severity":      severityNorm,
		"trustPenalty":  trustPenalty,
		"filterContext": filterBoost,
	}

	return score, marketImpliedRisk, components, rankReasonList(row, confidenceNorm, freshnessNorm, trustPenalty)
}

func decodeCursor(cursor string) int {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	text := strings.TrimPrefix(string(raw), "i:")
	n, err := strconv.Atoi(text)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func encodeCursor(offset int) string {
	if offset <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("i:%d", offset)))
}

func LoadFeedHistory(ctx context.Context, db *pgxpool.Pool) (FeedHistory, error) {
	var oldest, newest time.Time
	err := db.QueryRow(ctx, `SELECT coalesce(min(last_activity_at), now()), coalesce(max(last_activity_at), now()) FROM incidents`).Scan(&oldest, &newest)
	if err != nil {
		return FeedHistory{}, err
	}
	availableDays := int(math.Ceil(newest.Sub(oldest).Hours() / 24))
	if availableDays < 0 {
		availableDays = 0
	}
	return FeedHistory{AvailableDays: availableDays, OldestAt: oldest, NewestAt: newest}, nil
}

func windowDuration(window string) time.Duration {
	switch window {
	case "1h":
		return time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	case "1y":
		return 365 * 24 * time.Hour
	case "5y":
		return 5 * 365 * 24 * time.Hour
	case "20y":
		return 20 * 365 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func requestedWindowDays(f QueueFilters) int {
	if f.Window == "custom" {
		if !f.From.IsZero() && !f.To.IsZero() {
			from, to := f.From, f.To
			if to.Before(from) {
				from, to = to, from
			}
			return int(math.Ceil(to.Sub(from).Hours() / 24))
		}
		return 0
	}
	return int(math.Ceil(windowDuration(f.Window).Hours() / 24))
}

func LoadRiskFeed(ctx context.Context, db *pgxpool.Pool, f QueueFilters, limit int, cursor, mode string) (FeedPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	offset := decodeCursor(cursor)
	where, args := whereClause(f)
	q := `SELECT i.id,i.primary_symbol,i.status,i.severity_band,coalesce(i.priority_score,0),coalesce(i.composite_risk,0),
coalesce(i.escalation_probability,0),coalesce(i.confidence,0),coalesce(i.trust_state,'stable'),coalesce(i.model_version,''),
coalesce(i.top_driver_1,''),coalesce(i.top_driver_2,''),i.last_activity_at,coalesce(m.country_code,'XX'),coalesce(m.region,''),
coalesce(m.industry,''),coalesce(m.sector,''),coalesce(m.venue,'')
FROM incidents i
LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol
WHERE i.status in ('open','ack') AND ` + where + `
ORDER BY i.last_activity_at DESC, i.id ASC
LIMIT 500`

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return FeedPage{}, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	ranked := make([]feedRanked, 0, 128)
	for rows.Next() {
		var r feedRow
		if err := rows.Scan(
			&r.ID, &r.Symbol, &r.Status, &r.SeverityBand, &r.PriorityScore, &r.CompositeRisk,
			&r.EscalationProbability, &r.Confidence, &r.TrustState, &r.ModelVersion,
			&r.TopDriver1, &r.TopDriver2, &r.LastActivityAt, &r.CountryCode, &r.Region,
			&r.Industry, &r.Sector, &r.Venue,
		); err != nil {
			continue
		}

		score, marketImplied, components, reason := rankFeedRow(r, f, now, mode)
		ranked = append(ranked, feedRanked{
			row:               r,
			rankScore:         score,
			marketImpliedRisk: marketImplied,
			components:        components,
			rankReason:        reason,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].rankScore == ranked[j].rankScore {
			return ranked[i].row.ID < ranked[j].row.ID
		}
		return ranked[i].rankScore > ranked[j].rankScore
	})

	total := len(ranked)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	nextCursor := ""
	if end < total {
		nextCursor = encodeCursor(end)
	}

	windowed := ranked[offset:end]
	out := make([]map[string]any, 0, len(windowed))
	for i, item := range windowed {
		drivers := []string{}
		if item.row.TopDriver1 != "" {
			drivers = append(drivers, item.row.TopDriver1)
		}
		if item.row.TopDriver2 != "" {
			drivers = append(drivers, item.row.TopDriver2)
		}

		out = append(out, map[string]any{
			"rank":                  offset + i + 1,
			"id":                    item.row.ID,
			"incident_id":           item.row.ID,
			"symbol":                item.row.Symbol,
			"status":                item.row.Status,
			"severityBand":          item.row.SeverityBand,
			"severity":              item.row.SeverityBand,
			"priorityScore":         item.row.PriorityScore,
			"compositeRisk":         item.row.CompositeRisk,
			"composite":             item.row.CompositeRisk,
			"escalationProbability": item.row.EscalationProbability,
			"escalation_prob":       item.row.EscalationProbability,
			"confidence":            item.row.Confidence,
			"trustState":            item.row.TrustState,
			"modelVersion":          item.row.ModelVersion,
			"countryCode":           item.row.CountryCode,
			"country_code":          item.row.CountryCode,
			"region":                item.row.Region,
			"industry":              item.row.Industry,
			"sector":                item.row.Sector,
			"venue":                 item.row.Venue,
			"topDrivers":            drivers,
			"lastActivityAt":        item.row.LastActivityAt,
			"timestamp":             item.row.LastActivityAt,
			"rankScore":             item.rankScore,
			"rank_score":            item.rankScore,
			"marketImpliedRisk":     item.marketImpliedRisk,
			"rankComponents":        item.components,
			"rankReason":            item.rankReason,
			"rank_reason":           item.rankReason,
		})
	}

	history, historyErr := LoadFeedHistory(ctx, db)
	if historyErr != nil {
		history = FeedHistory{}
	}

	return FeedPage{
		Items:          out,
		NextCursor:     nextCursor,
		ServerTime:     now,
		History:        history,
		HistoryLimited: history.AvailableDays > 0 && requestedWindowDays(f) > history.AvailableDays,
	}, nil
}
