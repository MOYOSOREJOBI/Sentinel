package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentinel/internal/config"
	"sentinel/internal/db"
	"sentinel/internal/healthcheck"
	"sentinel/internal/kafka"
	"sentinel/internal/logging"
	"sentinel/internal/metrics"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Tick struct {
	EventID   string    `json:"event_id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	EventTime time.Time `json:"event_time"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8081)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.New("aggregator")
	cfg := config.Load("aggregator")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("aggregator: invalid config: %v", err)
	}
	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("aggregator: failed to connect to database: %v", err)
	}
	defer pool.Close()
	client, err := kafka.New(cfg.KafkaBrokers, "aggregator", "raw.ticks")
	if err != nil {
		log.Fatalf("aggregator: failed to create kafka consumer: %v", err)
	}
	defer client.Close()
	prod, err := kafka.New(cfg.KafkaBrokers, "")
	if err != nil {
		log.Fatalf("aggregator: failed to create kafka producer: %v", err)
	}
	defer prod.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"aggregator","version":"dev","links":["/healthz","/readyz","/metrics","/docs"]}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, probeCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer probeCancel()
		if err := pool.Ping(probeCtx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		if err := client.Ping(probeCtx); err != nil {
			http.Error(w, "kafka consumer not ready", http.StatusServiceUnavailable)
			return
		}
		if err := prod.Ping(probeCtx); err != nil {
			http.Error(w, "kafka producer not ready", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", metrics.Handler)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: metrics.Wrap("aggregator", mux), ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() {
		logger.Info("starting health server", map[string]any{"addr": cfg.HTTPAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("aggregator: health server error: %v", err)
		}
	}()

	go func() {
		for {
			fetches := client.PollFetches(ctx)
			if ctx.Err() != nil {
				return
			}
			fetches.EachRecord(func(rec *kgo.Record) {
				metrics.IncKafkaConsumed("aggregator", rec.Topic)
				var t Tick
				if err := json.Unmarshal(rec.Value, &t); err != nil || t.Symbol == "" {
					logger.Error("malformed tick payload", map[string]any{"error": fmt.Sprintf("%v", err), "topic": rec.Topic, "partition": rec.Partition})
					return
				}
				if t.EventTime.IsZero() {
					t.EventTime = time.Now().UTC()
				}

				tx, err := pool.Begin(ctx)
				if err != nil {
					return
				}
				defer tx.Rollback(ctx)

				ct, err := tx.Exec(ctx, `INSERT INTO raw_ticks(event_id,symbol,price,volume,event_time) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, t.EventID, t.Symbol, t.Price, t.Volume, t.EventTime)
				if err != nil {
					logger.Error("failed to insert raw tick", map[string]any{"error": err.Error(), "symbol": t.Symbol})
					return
				}
				if ct.RowsAffected() == 0 {
					logger.Info("duplicate raw tick no-op", map[string]any{"symbol": t.Symbol, "event_id": t.EventID})
					return
				} // duplicate event: no side effects

				bucket1s := t.EventTime.Truncate(time.Second)
				bucket1m := t.EventTime.Truncate(time.Minute)
				h1 := sha256.Sum256([]byte(t.Symbol + bucket1s.Format(time.RFC3339Nano) + "1s"))
				k1 := hex.EncodeToString(h1[:])
				hm := sha256.Sum256([]byte(t.Symbol + bucket1m.Format(time.RFC3339Nano) + "1m"))
				km := hex.EncodeToString(hm[:])

				start := time.Now()
				if _, err := tx.Exec(ctx, `
INSERT INTO candles(idempotency_key,symbol,bucket,interval,open,high,low,close,volume,open_event_time,close_event_time)
VALUES($1,$2,$3,'1s',$4,$4,$4,$4,$5,$6,$6)
ON CONFLICT (idempotency_key,bucket) DO UPDATE SET
	high=GREATEST(candles.high,EXCLUDED.high),
	low=LEAST(candles.low,EXCLUDED.low),
	open=CASE WHEN EXCLUDED.open_event_time < candles.open_event_time THEN EXCLUDED.open ELSE candles.open END,
	open_event_time=LEAST(candles.open_event_time,EXCLUDED.open_event_time),
	close=CASE WHEN EXCLUDED.close_event_time >= candles.close_event_time THEN EXCLUDED.close ELSE candles.close END,
	close_event_time=GREATEST(candles.close_event_time,EXCLUDED.close_event_time),
	volume=candles.volume+EXCLUDED.volume`, k1, t.Symbol, bucket1s, t.Price, t.Volume, t.EventTime); err != nil {
					return
				}

				metrics.ObserveDBQuery("aggregator", "upsert candles 1s", time.Since(start))
				start = time.Now()
				if _, err := tx.Exec(ctx, `
INSERT INTO candles(idempotency_key,symbol,bucket,interval,open,high,low,close,volume,open_event_time,close_event_time)
VALUES($1,$2,$3,'1m',$4,$4,$4,$4,$5,$6,$6)
ON CONFLICT (idempotency_key,bucket) DO UPDATE SET
	high=GREATEST(candles.high,EXCLUDED.high),
	low=LEAST(candles.low,EXCLUDED.low),
	open=CASE WHEN EXCLUDED.open_event_time < candles.open_event_time THEN EXCLUDED.open ELSE candles.open END,
	open_event_time=LEAST(candles.open_event_time,EXCLUDED.open_event_time),
	close=CASE WHEN EXCLUDED.close_event_time >= candles.close_event_time THEN EXCLUDED.close ELSE candles.close END,
	close_event_time=GREATEST(candles.close_event_time,EXCLUDED.close_event_time),
	volume=candles.volume+EXCLUDED.volume`, km, t.Symbol, bucket1m, t.Price, t.Volume, t.EventTime); err != nil {
					return
				}

				metrics.ObserveDBQuery("aggregator", "upsert candles 1m", time.Since(start))
				if err := tx.Commit(ctx); err != nil {
					logger.Error("failed to commit aggregation tx", map[string]any{"error": err.Error(), "symbol": t.Symbol})
					return
				}
				if err := kafka.ProduceJSON(ctx, prod, "derived.candles", t.Symbol, map[string]any{"symbol": t.Symbol, "bucket": bucket1s, "price": t.Price, "volume": t.Volume, "event_time": t.EventTime, "version": "v2"}); err == nil {
					metrics.IncKafkaProduced("aggregator", "derived.candles")
				}
			})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
