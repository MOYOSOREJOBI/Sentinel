//go:build seedusers

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type instrumentSeed struct {
	ID          string
	Venue       string
	AssetClass  string
	Country     string
	CountryCode string
	CountryName string
	Region      string
	Sector      string
	Industry    string
	Timezone    string
	BenchmarkID string
}

type sectorSeedProfile struct {
	Prefix      string
	Venue       string
	Country     string
	CountryCode string
	CountryName string
	Region      string
	Sector      string
	Industry    string
	Timezone    string
	BenchmarkID string
}

func simulatorUniverseSeeds() []instrumentSeed {
	base := []instrumentSeed{
		{"AAPL", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Consumer Electronics", "America/New_York", "QQQ"},
		{"MSFT", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "QQQ"},
		{"GOOGL", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Internet Platforms", "America/New_York", "QQQ"},
		{"AMZN", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "E-Commerce", "America/New_York", "XLY"},
		{"META", "NASDAQ", "equity", "US", "US", "United States", "North America", "Communications", "Platforms", "America/New_York", "XLC"},
		{"NVDA", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "QQQ"},
		{"TSLA", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Automotive", "America/New_York", "XLY"},
		{"JPM", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Banking", "America/New_York", "XLF"},
		{"BAC", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Banking", "America/New_York", "XLF"},
		{"WFC", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Banking", "America/New_York", "XLF"},
		{"XOM", "NYSE", "equity", "US", "US", "United States", "North America", "Energy", "Integrated Oil & Gas", "America/New_York", "XLE"},
		{"CVX", "NYSE", "equity", "US", "US", "United States", "North America", "Energy", "Integrated Oil & Gas", "America/New_York", "XLE"},
		{"COP", "NYSE", "equity", "US", "US", "United States", "North America", "Energy", "Exploration & Production", "America/New_York", "XLE"},
		{"UNH", "NYSE", "equity", "US", "US", "United States", "North America", "Healthcare", "Managed Care", "America/New_York", "XLV"},
		{"PFE", "NYSE", "equity", "US", "US", "United States", "North America", "Healthcare", "Pharmaceuticals", "America/New_York", "XLV"},
		{"JNJ", "NYSE", "equity", "US", "US", "United States", "North America", "Healthcare", "Pharmaceuticals", "America/New_York", "XLV"},
		{"MRK", "NYSE", "equity", "US", "US", "United States", "North America", "Healthcare", "Pharmaceuticals", "America/New_York", "XLV"},
		{"V", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Payments", "America/New_York", "XLF"},
		{"MA", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Payments", "America/New_York", "XLF"},
		{"PYPL", "NASDAQ", "equity", "US", "US", "United States", "North America", "Financials", "Payments", "America/New_York", "XLF"},
		{"DIS", "NYSE", "equity", "US", "US", "United States", "North America", "Communications", "Media", "America/New_York", "XLC"},
		{"NFLX", "NASDAQ", "equity", "US", "US", "United States", "North America", "Communications", "Streaming", "America/New_York", "XLC"},
		{"KO", "NYSE", "equity", "US", "US", "United States", "North America", "Consumer", "Beverages", "America/New_York", "XLP"},
		{"PEP", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Beverages", "America/New_York", "XLP"},
		{"COST", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Retail", "America/New_York", "XLP"},
		{"WMT", "NYSE", "equity", "US", "US", "United States", "North America", "Consumer", "Retail", "America/New_York", "XLP"},
		{"TGT", "NYSE", "equity", "US", "US", "United States", "North America", "Consumer", "Retail", "America/New_York", "XLP"},
		{"INTC", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "QQQ"},
		{"AMD", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "QQQ"},
		{"AVGO", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "QQQ"},
		{"ORCL", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "XLK"},
		{"ADBE", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "XLK"},
		{"CRM", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "XLK"},
		{"QCOM", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "XLK"},
		{"IBM", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "IT Services", "America/New_York", "XLK"},
		{"GE", "NYSE", "equity", "US", "US", "United States", "North America", "Industrials", "Industrial Conglomerates", "America/New_York", "XLI"},
		{"CAT", "NYSE", "equity", "US", "US", "United States", "North America", "Industrials", "Machinery", "America/New_York", "XLI"},
		{"BA", "NYSE", "equity", "US", "US", "United States", "North America", "Industrials", "Aerospace & Defense", "America/New_York", "XLI"},
		{"NKE", "NYSE", "equity", "US", "US", "United States", "North America", "Consumer", "Apparel", "America/New_York", "XLY"},
		{"SBUX", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Restaurants", "America/New_York", "XLY"},
		{"PLTR", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "XLK"},
		{"SNOW", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "Cloud Infrastructure", "America/New_York", "XLK"},
		{"SHOP", "NYSE", "equity", "CA", "CA", "Canada", "North America", "Technology", "E-Commerce Platforms", "America/Toronto", "XIT"},
		{"UBER", "NYSE", "equity", "US", "US", "United States", "North America", "Technology", "Mobility Platforms", "America/New_York", "XLK"},
		{"ABNB", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Travel Platforms", "America/New_York", "XLY"},
		{"BTC-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"ETH-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"SOL-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"GLD", "NYSEARCA", "fund", "US", "US", "United States", "North America", "Commodities", "Gold", "America/New_York", "GLD"},
		{"TLT", "NASDAQ", "fund", "US", "US", "United States", "North America", "Fixed Income", "Treasuries", "America/New_York", "TLT"},
	}

	profiles := []sectorSeedProfile{
		{Prefix: "TECH", Venue: "NASDAQ", Country: "US", CountryCode: "US", CountryName: "United States", Region: "North America", Sector: "Technology", Industry: "Software", Timezone: "America/New_York", BenchmarkID: "QQQ"},
		{Prefix: "FIN", Venue: "NYSE", Country: "US", CountryCode: "US", CountryName: "United States", Region: "North America", Sector: "Financials", Industry: "Diversified Financials", Timezone: "America/New_York", BenchmarkID: "XLF"},
		{Prefix: "HC", Venue: "NYSE", Country: "US", CountryCode: "US", CountryName: "United States", Region: "North America", Sector: "Healthcare", Industry: "Medical Devices", Timezone: "America/New_York", BenchmarkID: "XLV"},
		{Prefix: "IND", Venue: "XETRA", Country: "DE", CountryCode: "DE", CountryName: "Germany", Region: "Europe", Sector: "Industrials", Industry: "Capital Goods", Timezone: "Europe/Berlin", BenchmarkID: "DAX"},
		{Prefix: "CONS", Venue: "TSE", Country: "JP", CountryCode: "JP", CountryName: "Japan", Region: "Asia", Sector: "Consumer", Industry: "Retail", Timezone: "Asia/Tokyo", BenchmarkID: "NKY"},
		{Prefix: "ENERGY", Venue: "TADAWUL", Country: "SA", CountryCode: "SA", CountryName: "Saudi Arabia", Region: "Middle East", Sector: "Energy", Industry: "Integrated Oil & Gas", Timezone: "Asia/Riyadh", BenchmarkID: "TASI"},
		{Prefix: "UTIL", Venue: "LSE", Country: "GB", CountryCode: "GB", CountryName: "United Kingdom", Region: "Europe", Sector: "Utilities", Industry: "Power Utilities", Timezone: "Europe/London", BenchmarkID: "FTSE"},
		{Prefix: "REIT", Venue: "ASX", Country: "AU", CountryCode: "AU", CountryName: "Australia", Region: "Oceania", Sector: "Real Estate", Industry: "REITs", Timezone: "Australia/Sydney", BenchmarkID: "AS51"},
		{Prefix: "MAT", Venue: "B3", Country: "BR", CountryCode: "BR", CountryName: "Brazil", Region: "Latin America", Sector: "Materials", Industry: "Metals & Mining", Timezone: "America/Sao_Paulo", BenchmarkID: "IBOV"},
		{Prefix: "COMM", Venue: "SGX", Country: "SG", CountryCode: "SG", CountryName: "Singapore", Region: "Asia", Sector: "Communications", Industry: "Telecom Services", Timezone: "Asia/Singapore", BenchmarkID: "STI"},
	}

	out := append([]instrumentSeed{}, base...)
	for _, profile := range profiles {
		for i := 1; i <= 100; i++ {
			out = append(out, instrumentSeed{
				ID:          fmt.Sprintf("%s-%03d", profile.Prefix, i),
				Venue:       profile.Venue,
				AssetClass:  "equity",
				Country:     profile.Country,
				CountryCode: profile.CountryCode,
				CountryName: profile.CountryName,
				Region:      profile.Region,
				Sector:      profile.Sector,
				Industry:    profile.Industry,
				Timezone:    profile.Timezone,
				BenchmarkID: profile.BenchmarkID,
			})
		}
	}
	return out
}

func severityBandForComposite(v float64) string {
	switch {
	case v >= 0.85:
		return "critical"
	case v >= 0.65:
		return "high"
	case v >= 0.35:
		return "elevated"
	default:
		return "stable"
	}
}

func clamp(min, value, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func demoIncidentSeeds(universe []instrumentSeed) []instrumentSeed {
	prefixes := []string{"CONS-", "REIT-", "UTIL-", "COMM-", "IND-", "MAT-", "ENERGY-", "TECH-", "FIN-", "HC-"}
	out := make([]instrumentSeed, 0, 200)
	for _, prefix := range prefixes {
		picked := 0
		for _, item := range universe {
			if !strings.HasPrefix(item.ID, prefix) {
				continue
			}
			out = append(out, item)
			picked++
			if picked == 20 {
				break
			}
		}
	}
	return out
}

func seedDemoReadModels(ctx context.Context, pool *pgxpool.Pool, universe []instrumentSeed) error {
	now := time.Now().UTC().Truncate(time.Minute)

	for idx, item := range demoIncidentSeeds(universe) {
		minutesAgo := (idx * 7) % (23 * 60)
		lastActivity := now.Add(-time.Duration(minutesAgo) * time.Minute)
		composite := 0.18 + float64((idx*7)%68)/100
		if idx%17 == 0 {
			composite += 0.08
		}
		if item.ID == "CONS-041" {
			composite = 0.94
		}
		composite = clamp(0.12, composite, 0.94)
		escalation := clamp(0.05, composite*0.82+float64((idx%9))*0.013, 0.97)
		confidence := clamp(0.71, 0.72+float64((idx*5)%25)/100, 0.97)
		severity := severityBandForComposite(composite)
		status := "open"
		if idx%5 == 0 {
			status = "ack"
		}
		featureHash := fmt.Sprintf("seed-demo:incident:%s", item.ID)
		artifactHash := fmt.Sprintf("seed-demo:artifact:%s", item.ID)

		if _, err := pool.Exec(ctx, `
INSERT INTO incidents (
  primary_symbol, status, severity_band, priority_score, composite_risk, escalation_probability, confidence,
  trust_state, feature_snapshot_hash, feature_set_version, model_version, calibration_version,
  top_driver_1, top_driver_2, top_driver_3, driver_payload, started_at, last_activity_at, owner_name, created_at, updated_at
)
SELECT
  $1, $2, $3, $4, $4, $5, $6,
  'stable', $7, 'v2', 'quant-v1-2026-03-01', 'cal-v2',
  $8, $9, 'geo', jsonb_build_object('seed', true, 'country', $10::text, 'sector', $11::text),
  $12, $12, '', $12, $12
WHERE NOT EXISTS (
  SELECT 1 FROM incidents WHERE feature_snapshot_hash = $7
)`,
			item.ID, status, severity, composite, escalation, confidence, featureHash,
			"watch", "Seeded demo coverage", item.CountryCode, item.Sector, lastActivity); err != nil {
			return err
		}

		if _, err := pool.Exec(ctx, `
INSERT INTO scores (
  idempotency_key, symbol, ts, score, severity, explanation, replay_run_id,
  raw_anomaly_score, normalized_anomaly_score, escalation_probability, priority_score, composite_risk,
  feature_snapshot_hash, feature_set_version, model_version, calibration_version, explanation_payload,
  scoring_run_id, produced_at, artifact_hash, model_artifact_hash
)
VALUES (
  $1, $2, $3, $4, $5, $6, '',
  $7, $8, $9, $4, $4,
  $10, 'v2', 'quant-v1-2026-03-01', 'cal-v2', jsonb_build_object('seed', true, 'symbol', $2::text),
  gen_random_uuid(), $11, $12, $12
)
ON CONFLICT (idempotency_key, ts) DO NOTHING`,
			fmt.Sprintf("seed-demo:score:%s", item.ID), item.ID, lastActivity, composite, severity, "Seeded demo score",
			composite-0.04, composite, escalation, featureHash, lastActivity.Add(10*time.Second), artifactHash); err != nil {
			return err
		}
	}

	for step := 0; step < 35*24; step++ {
		ts := now.Add(-time.Duration(step) * time.Hour)
		sinWave := math.Sin(float64(step) / 8)
		cosWave := math.Cos(float64(step) / 13)
		composite := clamp(0.18, 0.48+0.24*sinWave+0.12*cosWave, 0.93)
		if step%29 == 0 {
			composite = clamp(0.18, composite+0.18, 0.95)
		}
		escalation := clamp(0.03, composite*0.78+0.04*math.Cos(float64(step)/5), 0.96)
		severity := severityBandForComposite(composite)
		priceBase := 118 + 14*math.Sin(float64(step)/6) + 9*math.Cos(float64(step)/11)
		open := clamp(1, priceBase+1.4*math.Sin(float64(step)/4), 9999)
		close := clamp(1, priceBase+1.1*math.Cos(float64(step)/7), 9999)
		high := clamp(open, math.Max(open, close)+2.6+math.Mod(float64(step), 4), 9999)
		low := clamp(0.01, math.Min(open, close)-2.1-math.Mod(float64(step), 3), high)
		volume := 12000 + math.Mod(float64(step*step), 7000)

		if _, err := pool.Exec(ctx, `
INSERT INTO scores (
  idempotency_key, symbol, ts, score, severity, explanation, replay_run_id,
  raw_anomaly_score, normalized_anomaly_score, escalation_probability, priority_score, composite_risk,
  feature_snapshot_hash, feature_set_version, model_version, calibration_version, explanation_payload,
  scoring_run_id, produced_at, artifact_hash, model_artifact_hash
)
VALUES (
  $1, 'CONS-041', $2, $3, $4, 'Seeded demo time-series', '',
  $5, $3, $6, $3, $3,
  $7, 'v2', 'quant-v1-2026-03-01', 'cal-v2', jsonb_build_object('seed', true, 'series', true),
  gen_random_uuid(), $8, $9, $9
)
ON CONFLICT (idempotency_key, ts) DO NOTHING`,
			fmt.Sprintf("seed-demo:score-series:CONS-041:%03d", step), ts, composite, severity, composite-0.05, escalation,
			fmt.Sprintf("seed-demo:series:CONS-041:%03d", step), ts.Add(15*time.Second), fmt.Sprintf("seed-demo:artifact:CONS-041:%03d", step)); err != nil {
			return err
		}

		if _, err := pool.Exec(ctx, `
INSERT INTO candles(idempotency_key, symbol, bucket, interval, open, high, low, close, volume, replay_run_id, open_event_time, close_event_time)
VALUES ($1, 'CONS-041', $2, '1m', $3, $4, $5, $6, $7, '', $2, $2)
ON CONFLICT (idempotency_key, bucket) DO NOTHING`,
			fmt.Sprintf("seed-demo:candle-series:CONS-041:%03d", step), ts, open, high, low, close, volume); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	users := []struct{ E, R string }{{"admin@sentinel.local", "admin"}, {"analyst@sentinel.local", "analyst"}, {"viewer@sentinel.local", "viewer"}}
	for _, u := range users {
		h, _ := bcrypt.GenerateFromPassword([]byte("Sentinel#123"), bcrypt.DefaultCost)
		_, err = pool.Exec(ctx, `INSERT INTO users(email,password_hash,role) VALUES($1,$2,$3) ON CONFLICT (email) DO UPDATE SET password_hash=$2, role=$3`, u.E, string(h), u.R)
		if err != nil {
			panic(err)
		}
	}

	seeds := append(simulatorUniverseSeeds(),
		instrumentSeed{"BABA", "NYSE", "equity", "CN", "CN", "China", "Asia", "Consumer", "E-Commerce", "Asia/Shanghai", "MCHI"},
		instrumentSeed{"SAP", "XETRA", "equity", "DE", "DE", "Germany", "Europe", "Technology", "Enterprise Software", "Europe/Berlin", "DAX"},
	)

	for _, s := range seeds {
		_, err = pool.Exec(ctx, `
INSERT INTO instrument_metadata(instrument_id,venue,asset_class,country,region,sector,industry,benchmark_id,country_code,country_name,timezone)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (instrument_id) DO UPDATE SET
 venue=excluded.venue,
 asset_class=excluded.asset_class,
 country=excluded.country,
 region=excluded.region,
 sector=excluded.sector,
 industry=excluded.industry,
 benchmark_id=excluded.benchmark_id,
 country_code=excluded.country_code,
 country_name=excluded.country_name,
 timezone=excluded.timezone
`, s.ID, s.Venue, s.AssetClass, s.Country, s.Region, s.Sector, s.Industry, s.BenchmarkID, s.CountryCode, s.CountryName, s.Timezone)
		if err != nil {
			panic(err)
		}
	}

	if err := seedDemoReadModels(ctx, pool, seeds); err != nil {
		panic(err)
	}

	for _, wl := range []struct {
		Owner string
		Name  string
		Items []string
	}{
		{Owner: "analyst@sentinel.local", Name: "Macro Risk", Items: []string{"AAPL", "MSFT", "TSLA", "BTC-USD"}},
		{Owner: "viewer@sentinel.local", Name: "Global Watch", Items: []string{"SAP", "BABA", "JPM", "ETH-USD"}},
	} {
		var id string
		err = pool.QueryRow(ctx, `INSERT INTO watchlists(owner_user,name) VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING id`, wl.Owner, wl.Name).Scan(&id)
		if err != nil {
			_ = pool.QueryRow(ctx, `SELECT id::text FROM watchlists WHERE owner_user=$1 AND name=$2 ORDER BY created_at DESC LIMIT 1`, wl.Owner, wl.Name).Scan(&id)
		}
		if id == "" {
			continue
		}
		for _, item := range wl.Items {
			_, _ = pool.Exec(ctx, `INSERT INTO watchlist_items(watchlist_id,instrument_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, item)
		}
	}

	_, _ = pool.Exec(ctx, `INSERT INTO user_preferences(owner_user, preferred_mode, preferred_region, preferred_industry, preferred_locale) VALUES($1,$2,$3,$4,$5) ON CONFLICT (owner_user) DO UPDATE SET preferred_mode=excluded.preferred_mode, preferred_region=excluded.preferred_region, preferred_industry=excluded.preferred_industry, preferred_locale=excluded.preferred_locale, updated_at=now()`, "analyst@sentinel.local", "analyst", "North America", "Technology", "en")
	_, _ = pool.Exec(ctx, `INSERT INTO user_preferences(owner_user, preferred_mode, preferred_region, preferred_industry, preferred_locale) VALUES($1,$2,$3,$4,$5) ON CONFLICT (owner_user) DO UPDATE SET preferred_mode=excluded.preferred_mode, preferred_region=excluded.preferred_region, preferred_industry=excluded.preferred_industry, preferred_locale=excluded.preferred_locale, updated_at=now()`, "viewer@sentinel.local", "viewer", "Europe", "Technology", "fr")

	fmt.Println("users, instruments, watchlists, and preferences seeded")
}
