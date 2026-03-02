package main

import (
	"context"
	"fmt"
	"os"

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

	seeds := []instrumentSeed{
		{"AAPL", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Consumer Electronics", "America/New_York", "QQQ"},
		{"MSFT", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Software", "America/New_York", "QQQ"},
		{"TSLA", "NASDAQ", "equity", "US", "US", "United States", "North America", "Consumer", "Automotive", "America/New_York", "QQQ"},
		{"NVDA", "NASDAQ", "equity", "US", "US", "United States", "North America", "Technology", "Semiconductors", "America/New_York", "QQQ"},
		{"BTC-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"ETH-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"SOL-USD", "CRYPTO", "crypto", "Global", "GB", "Global", "Global", "Digital Assets", "Crypto", "UTC", "CRYPTO"},
		{"JPM", "NYSE", "equity", "US", "US", "United States", "North America", "Financials", "Banking", "America/New_York", "SPY"},
		{"BABA", "NYSE", "equity", "CN", "CN", "China", "Asia", "Consumer", "E-Commerce", "Asia/Shanghai", "MCHI"},
		{"SAP", "XETRA", "equity", "DE", "DE", "Germany", "Europe", "Technology", "Enterprise Software", "Europe/Berlin", "DAX"},
	}

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

	_, _ = pool.Exec(ctx, `INSERT INTO user_preferences(owner_user, preferred_mode, preferred_region, preferred_industry) VALUES($1,$2,$3,$4) ON CONFLICT (owner_user) DO UPDATE SET preferred_mode=excluded.preferred_mode, preferred_region=excluded.preferred_region, preferred_industry=excluded.preferred_industry, updated_at=now()`, "analyst@sentinel.local", "analyst", "North America", "Technology")
	_, _ = pool.Exec(ctx, `INSERT INTO user_preferences(owner_user, preferred_mode, preferred_region, preferred_industry) VALUES($1,$2,$3,$4) ON CONFLICT (owner_user) DO UPDATE SET preferred_mode=excluded.preferred_mode, preferred_region=excluded.preferred_region, preferred_industry=excluded.preferred_industry, updated_at=now()`, "viewer@sentinel.local", "viewer", "Europe", "Technology")

	fmt.Println("users, instruments, watchlists, and preferences seeded")
}
