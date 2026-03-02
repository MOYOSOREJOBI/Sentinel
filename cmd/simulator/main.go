package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/ksuid"
	"sentinel/internal/config"
	"sentinel/internal/healthcheck"
	"sentinel/internal/kafka"
	"sentinel/internal/logging"
	"sentinel/internal/metrics"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8086)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.New("simulator")
	cfg := config.Load("simulator")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("simulator: invalid config: %v", err)
	}

	k, err := kafka.New(cfg.KafkaBrokers, "")
	if err != nil {
		log.Fatalf("simulator: failed to create kafka producer: %v", err)
	}
	defer k.Close()

	symbols := parseSymbols(os.Getenv("SIM_SYMBOLS"))
	if len(symbols) == 0 {
		symbols = defaultSymbols()
	}
	prices := map[string]float64{"AAPL": 190, "MSFT": 420, "TSLA": 230, "NVDA": 780, "BTC-USD": 62000}
	interval := 500 * time.Millisecond
	if ms, err := strconv.Atoi(os.Getenv("SIM_INTERVAL_MS")); err == nil && ms > 0 {
		interval = time.Duration(ms) * time.Millisecond
	}
	anomalyEvery := 50
	if n, err := strconv.Atoi(os.Getenv("SIM_ANOMALY_EVERY")); err == nil && n > 0 {
		anomalyEvery = n
	}

	logger.Info("starting simulator", map[string]any{"symbols": symbols, "interval": interval.String(), "anomaly_every": anomalyEvery})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"simulator","version":"dev","links":["/healthz","/readyz","/metrics","/docs"]}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, cancelProbe := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancelProbe()
		if err := k.Ping(probeCtx); err != nil {
			http.Error(w, "kafka producer not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", metrics.Handler)
	srv := &http.Server{Addr: healthcheckAddr(cfg.HTTPAddr, ":8086"), Handler: metrics.Wrap("simulator", mux), ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("simulator: health server error: %v", err)
		}
	}()

	ticks := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			logger.Info("shutting down", nil)
			return
		case <-ticker.C:
			for _, s := range symbols {
				delta := rand.Float64()*2 - 1
				if ticks > 0 && ticks%anomalyEvery == 0 {
					delta = delta * 15
				}
				prices[s] += delta
				if prices[s] < 0.01 {
					prices[s] = 0.01
				}
				ev := map[string]any{
					"event_id":   ksuid.New().String(),
					"symbol":     s,
					"price":      prices[s],
					"volume":     100 + rand.Float64()*20,
					"event_time": time.Now().UTC().Format(time.RFC3339Nano),
				}
				if err := kafka.ProduceJSON(ctx, k, "raw.ticks", s, ev); err != nil {
					logger.Error("produce tick failed", map[string]any{"error": err.Error(), "symbol": s})
				} else {
					metrics.IncKafkaProduced("simulator", "raw.ticks")
				}
				ticks++
			}
		}
	}
}

func parseSymbols(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func defaultSymbols() []string {
	base := []string{
		"AAPL", "MSFT", "GOOGL", "AMZN", "META", "NVDA", "TSLA", "JPM", "BAC", "WFC",
		"XOM", "CVX", "COP", "UNH", "PFE", "JNJ", "MRK", "V", "MA", "PYPL",
		"DIS", "NFLX", "KO", "PEP", "COST", "WMT", "TGT", "INTC", "AMD", "AVGO",
		"ORCL", "ADBE", "CRM", "QCOM", "IBM", "GE", "CAT", "BA", "NKE", "SBUX",
		"PLTR", "SNOW", "SHOP", "UBER", "ABNB", "BTC-USD", "ETH-USD", "SOL-USD", "GLD", "TLT",
	}
	out := append([]string{}, base...)
	sectors := []string{"TECH", "FIN", "HC", "IND", "CONS", "ENERGY", "UTIL", "REIT", "MAT", "COMM"}
	for _, sec := range sectors {
		for i := 1; i <= 100; i++ {
			out = append(out, fmt.Sprintf("%s-%03d", sec, i))
		}
	}
	return out
}

func healthcheckAddr(addr, fallback string) string {
	if strings.TrimSpace(addr) == "" {
		return fallback
	}
	return addr
}
