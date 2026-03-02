package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type QueueFilters struct {
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

type QueueRow struct {
	ID                int64   `json:"id"`
	Symbol            string  `json:"symbol"`
	Status            string  `json:"status"`
	SeverityBand      string  `json:"severityBand"`
	PriorityScore     float64 `json:"priorityScore"`
	CompositeRisk     float64 `json:"compositeRisk"`
	Confidence        float64 `json:"confidence"`
	Region            string  `json:"region"`
	CountryCode       string  `json:"countryCode"`
	Industry          string  `json:"industry"`
	RecommendedAction string  `json:"recommendedAction"`
}

func windowSQL(tw string) string {
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

func whereClause(f QueueFilters) (string, []any) {
	args := []any{}
	where := windowSQL(f.Window)
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
	add := func(col string, val any, skip bool) {
		if skip {
			return
		}
		args = append(args, val)
		where += fmt.Sprintf(" AND %s=$%d", col, len(args))
	}
	add("m.region", f.Region, f.Region == "")
	add("coalesce(m.country_code,'XX')", f.CountryCode, f.CountryCode == "")
	add("m.sector", f.Sector, f.Sector == "")
	add("m.industry", f.Industry, f.Industry == "")
	add("m.venue", f.Venue, f.Venue == "")
	add("i.primary_symbol", f.Symbol, f.Symbol == "")
	if strings.TrimSpace(f.Q) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(f.Q))+"%")
		idx := len(args)
		where += fmt.Sprintf(" AND (lower(i.primary_symbol) LIKE $%d OR lower(coalesce(m.country_code,'')) LIKE $%d OR lower(coalesce(m.region,'')) LIKE $%d OR lower(coalesce(m.sector,'')) LIKE $%d OR lower(coalesce(m.industry,'')) LIKE $%d OR lower(coalesce(m.venue,'')) LIKE $%d)", idx, idx, idx, idx, idx, idx)
	}
	return where, args
}

func LoadQueue(ctx context.Context, db *pgxpool.Pool, f QueueFilters) ([]QueueRow, error) {
	where, args := whereClause(f)
	q := `SELECT i.id,i.primary_symbol,i.status,i.severity_band,coalesce(i.priority_score,0),coalesce(i.composite_risk,0),coalesce(i.confidence,0),coalesce(m.region,''),coalesce(m.country_code,'XX'),coalesce(m.industry,''),coalesce(i.top_driver_1,'watch') FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE ` + where
	q += " ORDER BY i.priority_score DESC,i.last_activity_at DESC LIMIT 250"

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QueueRow{}
	for rows.Next() {
		var r QueueRow
		if rows.Scan(&r.ID, &r.Symbol, &r.Status, &r.SeverityBand, &r.PriorityScore, &r.CompositeRisk, &r.Confidence, &r.Region, &r.CountryCode, &r.Industry, &r.RecommendedAction) == nil {
			out = append(out, r)
		}
	}
	return out, nil
}
