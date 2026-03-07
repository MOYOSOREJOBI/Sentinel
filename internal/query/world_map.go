package query

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CountryAgg struct {
	CountryISO2      string  `json:"countryIso2"`
	CountryCode      string  `json:"countryCode"`
	CountryName      string  `json:"countryName"`
	IncidentCount    int     `json:"incidentCount"`
	AvgCompositeRisk float64 `json:"avgCompositeRisk"`
	MaxSafetyLevel   string  `json:"maxSafetyLevel"`
	TrustState       string  `json:"trustState"`
	TopIndustry      string  `json:"topIndustry,omitempty"`
}

type WorldMapSummary struct {
	Countries       []CountryAgg `json:"countries"`
	GeoEnriched     bool         `json:"geoEnriched"`
	MissingGeoCount int          `json:"missingGeoCount"`
}

func LoadWorldMap(ctx context.Context, db *pgxpool.Pool, f QueueFilters) ([]CountryAgg, error) {
	where, args := whereClause(f)

	q := `SELECT upper(m.country_code),coalesce(nullif(m.country_name,''),upper(m.country_code)),count(i.id),coalesce(avg(i.composite_risk),0),coalesce(max(i.severity_band),'Stable'),coalesce(max(i.trust_state),'healthy'),coalesce(max(m.industry),'')
FROM incidents i
JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol
WHERE ` + where + `
  AND char_length(coalesce(m.country_code,'')) = 2
  AND upper(m.country_code) <> 'XX'
  AND coalesce(nullif(m.country_name,''),'') <> ''
GROUP BY 1,2 ORDER BY 3 DESC LIMIT 250`
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CountryAgg{}
	for rows.Next() {
		var c CountryAgg
		if rows.Scan(&c.CountryCode, &c.CountryName, &c.IncidentCount, &c.AvgCompositeRisk, &c.MaxSafetyLevel, &c.TrustState, &c.TopIndustry) == nil {
			c.CountryISO2 = strings.ToUpper(strings.TrimSpace(c.CountryCode))
			c.CountryCode = c.CountryISO2
			out = append(out, c)
		}
	}
	return out, nil
}

func CountMissingGeo(ctx context.Context, db *pgxpool.Pool, f QueueFilters) (int, error) {
	where, args := whereClause(f)
	q := `SELECT count(i.id)
FROM incidents i
LEFT JOIN instrument_metadata m ON m.instrument_id=i.primary_symbol
WHERE ` + where + `
  AND (
    char_length(coalesce(m.country_code,'')) <> 2
    OR upper(coalesce(m.country_code,'')) = 'XX'
    OR coalesce(nullif(m.country_name,''),'') = ''
    OR coalesce(nullif(m.region,''),'') = ''
    OR coalesce(nullif(m.sector,''),'') = ''
    OR coalesce(nullif(m.industry,''),'') = ''
    OR coalesce(nullif(m.venue,''),'') = ''
  )`
	var missing int
	if err := db.QueryRow(ctx, q, args...).Scan(&missing); err != nil {
		return 0, err
	}
	return missing, nil
}

func LoadWorldMapSummary(ctx context.Context, db *pgxpool.Pool, f QueueFilters) (WorldMapSummary, error) {
	countries, err := LoadWorldMap(ctx, db, f)
	if err != nil {
		return WorldMapSummary{}, err
	}
	missing, err := CountMissingGeo(ctx, db, f)
	if err != nil {
		return WorldMapSummary{}, err
	}
	return WorldMapSummary{
		Countries:       countries,
		GeoEnriched:     missing == 0,
		MissingGeoCount: missing,
	}, nil
}
