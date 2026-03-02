package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CountryAgg struct {
	CountryCode      string  `json:"countryCode"`
	CountryName      string  `json:"countryName"`
	IncidentCount    int     `json:"incidentCount"`
	AvgCompositeRisk float64 `json:"avgCompositeRisk"`
	MaxSafetyLevel   string  `json:"maxSafetyLevel"`
	TrustState       string  `json:"trustState"`
	TopIndustry      string  `json:"topIndustry,omitempty"`
}

func LoadWorldMap(ctx context.Context, db *pgxpool.Pool, f QueueFilters) ([]CountryAgg, error) {
	where, args := whereClause(f)

	q := `SELECT coalesce(m.country_code,'XX'),coalesce(m.country_name,'Unknown'),count(i.id),coalesce(avg(i.composite_risk),0),coalesce(max(i.severity_band),'Stable'),coalesce(max(i.trust_state),'healthy'),coalesce(max(m.industry),'') FROM incidents i LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol WHERE ` + where + ` GROUP BY 1,2 ORDER BY 3 DESC LIMIT 250`
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CountryAgg{}
	for rows.Next() {
		var c CountryAgg
		if rows.Scan(&c.CountryCode, &c.CountryName, &c.IncidentCount, &c.AvgCompositeRisk, &c.MaxSafetyLevel, &c.TrustState, &c.TopIndustry) == nil {
			out = append(out, c)
		}
	}
	return out, nil
}
