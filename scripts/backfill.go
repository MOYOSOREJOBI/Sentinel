package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"sentinel/internal/db"
	qrm "sentinel/internal/query"
)

func main() {
	years := 20
	if raw := os.Getenv("YEARS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			log.Fatalf("invalid YEARS=%q", raw)
		}
		years = parsed
	}

	pgURL := os.Getenv("POSTGRES_URL")
	if pgURL == "" {
		pgURL = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := db.Connect(ctx, pgURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := qrm.BackfillHistory(ctx, pool, years); err != nil {
		log.Fatalf("backfill history: %v", err)
	}

	fmt.Printf("backfill complete: %d years\n", years)
}
