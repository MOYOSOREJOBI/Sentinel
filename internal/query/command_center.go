package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadCommandCenter(ctx context.Context, db *pgxpool.Pool, f QueueFilters) (map[string]any, error) {
	where, args := whereClause(f)

	var open, high int
	_ = db.QueryRow(ctx, `SELECT count(i.id) FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE i.status in ('open','ack') AND `+where, args...).Scan(&open)
	_ = db.QueryRow(ctx, `SELECT count(i.id) FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE i.status in ('open','ack') AND i.severity_band in ('high','critical') AND `+where, args...).Scan(&high)

	topIncidents := []map[string]any{}
	rows, err := db.Query(ctx, `SELECT i.id,i.primary_symbol,coalesce(i.priority_score,0),coalesce(i.composite_risk,0),i.severity_band FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE i.status in ('open','ack') AND `+where+` ORDER BY i.priority_score DESC LIMIT 5`, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var sym, sev string
			var p, c float64
			if rows.Scan(&id, &sym, &p, &c, &sev) == nil {
				topIncidents = append(topIncidents, map[string]any{"id": id, "symbol": sym, "priorityScore": p, "compositeRisk": c, "severityBand": sev})
			}
		}
	}

	countries := []map[string]any{}
	cr, err := db.Query(ctx, `SELECT coalesce(m.country_code,'XX'),count(i.id) FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE `+where+` GROUP BY 1 ORDER BY 2 DESC LIMIT 5`, args...)
	if err == nil {
		defer cr.Close()
		for cr.Next() {
			var code string
			var c int
			if cr.Scan(&code, &c) == nil {
				countries = append(countries, map[string]any{"countryCode": code, "incidentCount": c})
			}
		}
	}

	industries := []map[string]any{}
	ir, err := db.Query(ctx, `SELECT coalesce(m.industry,'Unknown'),count(i.id) FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE `+where+` GROUP BY 1 ORDER BY 2 DESC LIMIT 5`, args...)
	if err == nil {
		defer ir.Close()
		for ir.Next() {
			var ind string
			var c int
			if ir.Scan(&ind, &c) == nil {
				industries = append(industries, map[string]any{"industry": ind, "incidentCount": c})
			}
		}
	}

	return map[string]any{"timeWindow": f.Window, "openIncidents": open, "highRiskCount": high, "backlogDelta": high - open, "topIncidents": topIncidents, "topCountries": countries, "topIndustries": industries, "trust": map[string]any{"state": "stable"}}, nil
}
