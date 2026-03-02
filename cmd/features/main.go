package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sentinel/internal/config"
	"sentinel/internal/contracts"
	"sentinel/internal/db"
	"sentinel/internal/healthcheck"
	"sentinel/internal/kafka"
	"sentinel/internal/logging"
	"sentinel/internal/marketmath"
	"sentinel/internal/metrics"

	"github.com/twmb/franz-go/pkg/kgo"
)

const eps = 1e-9

type symbolState struct {
	LastPrice float64
	LastEvent time.Time

	Ret60     *marketmath.Ring
	Ret300    *marketmath.Ring
	Vol60     *marketmath.Ring
	Log300    *marketmath.Ring
	PriceHist *marketmath.Ring

	EWMA60  *marketmath.EWMA
	EWMA300 *marketmath.EWMA

	TotalEvents     float64
	LateEvents      float64
	OutOfOrder      float64
	DuplicateEvents float64
	MissingEvents   float64

	RecentAnomaly *marketmath.Ring
}

func newSymbolState() *symbolState {
	return &symbolState{
		Ret60:         marketmath.NewRing(60),
		Ret300:        marketmath.NewRing(300),
		Vol60:         marketmath.NewRing(60),
		Log300:        marketmath.NewRing(300),
		EWMA60:        marketmath.NewEWMA(0.94),
		EWMA300:       marketmath.NewEWMA(0.97),
		RecentAnomaly: marketmath.NewRing(300),
		PriceHist:     marketmath.NewRing(301),
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8082)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.New("features")
	cfg := config.Load("features")
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("features: invalid config: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("features: failed to connect to database: %v", err)
	}
	defer pool.Close()

	c, err := kafka.New(cfg.KafkaBrokers, "features", "derived.candles")
	if err != nil {
		log.Fatalf("features: failed to create kafka consumer: %v", err)
	}
	defer c.Close()

	p, err := kafka.New(cfg.KafkaBrokers, "")
	if err != nil {
		log.Fatalf("features: failed to create kafka producer: %v", err)
	}
	defer p.Close()

	states := map[string]*symbolState{}
	var mu sync.Mutex

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"features","version":"dev","links":["/healthz","/readyz","/metrics","/docs"]}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, probeCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer probeCancel()
		if err := pool.Ping(probeCtx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		if err := c.Ping(probeCtx); err != nil {
			http.Error(w, "kafka consumer not ready", http.StatusServiceUnavailable)
			return
		}
		if err := p.Ping(probeCtx); err != nil {
			http.Error(w, "kafka producer not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", metrics.Handler)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: metrics.Wrap("features", mux), ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() {
		logger.Info("starting health server", map[string]any{"addr": cfg.HTTPAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("features: health server error: %v", err)
		}
	}()

	go func() {
		for {
			f := c.PollFetches(ctx)
			if ctx.Err() != nil {
				return
			}
			f.EachRecord(func(rec *kgo.Record) {
				metrics.IncKafkaConsumed("features", rec.Topic)
				var m map[string]any
				if err := json.Unmarshal(rec.Value, &m); err != nil {
					return
				}
				symbol, _ := m["symbol"].(string)
				price, _ := m["price"].(float64)
				vol, _ := m["volume"].(float64)
				if symbol == "" || price <= 0 {
					return
				}
				eventAt := time.Now().UTC()
				if raw, ok := m["event_time"].(string); ok {
					if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
						eventAt = ts
					}
				}

				mu.Lock()
				st := states[symbol]
				if st == nil {
					st = newSymbolState()
					states[symbol] = st
				}
				payload, dq := computeFeatures(st, price, vol, eventAt)
				mu.Unlock()

				ordered := []string{"ret_1s", "ret_1m", "ret_5m", "log_ret_1s", "log_ret_1m", "log_ret_5m", "mean_ret_60", "std_ret_60", "z_ret_60", "mean_ret_300", "std_ret_300", "z_ret_300", "ewma_var_60", "ewma_var_300", "ewma_vol_60", "ewma_vol_300", "realized_var_300", "realized_vol_300", "mean_vol_60", "std_vol_60", "volume_ratio_60", "volume_surprise_60", "anomaly_density_300", "tick_gap_s", "staleness_score", "missingness_ratio", "out_of_order_ratio", "duplicate_ratio", "late_ratio", "dq_penalty"}
				hash, err := marketmath.StableFeatureHash(payload, ordered)
				if err != nil {
					return
				}
				msg := contracts.FeatureVector{Symbol: symbol, EventTime: eventAt, FeatureSetVersion: "v2", AllowedLatenessMS: 5000, WatermarkPolicyID: "wm_v1", FeatureSnapshotHash: hash, Payload: payload}
				pb, _ := json.Marshal(msg)
				id := symbol + hash[:16]
				start := time.Now()
				_, _ = pool.Exec(ctx, `INSERT INTO features(idempotency_key,symbol,ts,feature_hash,payload) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, id, symbol, eventAt, hash, pb)
				metrics.ObserveDBQuery("features", "insert features", time.Since(start))
				if err := kafka.ProduceJSON(ctx, p, "derived.features", symbol, map[string]any{"symbol": symbol, "payload": payload, "feature_hash": hash, "event_time": eventAt, "feature_set_version": "v2"}); err == nil {
					metrics.IncKafkaProduced("features", "derived.features")
				}
				if err := kafka.ProduceJSON(ctx, p, "dq.metrics", symbol, map[string]any{"symbol": symbol, "dq_penalty": dq, "event_time": eventAt}); err == nil {
					metrics.IncKafkaProduced("features", "dq.metrics")
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
	_ = srv.Shutdown(shutdownCtx)
}

func computeFeatures(st *symbolState, price, vol float64, eventAt time.Time) (map[string]float64, float64) {
	st.TotalEvents++
	if !st.LastEvent.IsZero() {
		if !eventAt.After(st.LastEvent) {
			st.OutOfOrder++
		}
		if eventAt.Equal(st.LastEvent) {
			st.DuplicateEvents++
		}
		if eventAt.Sub(st.LastEvent) > 2*time.Second {
			st.MissingEvents++
		}
		if eventAt.Before(st.LastEvent.Add(-5 * time.Second)) {
			st.LateEvents++
		}
	}

	ret := horizonReturn(st.PriceHist, price, 1)
	ret1m := horizonReturn(st.PriceHist, price, 60)
	ret5m := horizonReturn(st.PriceHist, price, 300)
	logRet := horizonLogReturn(st.PriceHist, price, 1)
	logRet1m := horizonLogReturn(st.PriceHist, price, 60)
	logRet5m := horizonLogReturn(st.PriceHist, price, 300)
	tickGap := 0.0
	if !st.LastEvent.IsZero() {
		tickGap = math.Max(0, eventAt.Sub(st.LastEvent).Seconds())
	}
	staleness := marketmath.Clip(tickGap/30.0, 0, 1)

	st.Ret60.Push(ret)
	st.Ret300.Push(ret)
	st.Vol60.Push(vol)
	st.Log300.Push(logRet)
	st.EWMA60.Update(ret)
	st.EWMA300.Update(ret)

	meanRet60 := st.Ret60.Mean()
	stdRet60 := st.Ret60.Std()
	meanRet300 := st.Ret300.Mean()
	stdRet300 := st.Ret300.Std()
	meanVol60 := st.Vol60.Mean()
	stdVol60 := st.Vol60.Std()
	z60 := marketmath.ZScore(ret, meanRet60, stdRet60)
	z300 := marketmath.ZScore(ret, meanRet300, stdRet300)
	volSurprise := marketmath.VolumeSurprise(vol, meanVol60, stdVol60)
	anomaly := 0.0
	if math.Abs(z60) > 2 || math.Abs(volSurprise) > 2 {
		anomaly = 1
	}
	st.RecentAnomaly.Push(anomaly)

	lateRatio := ratio(st.LateEvents, st.TotalEvents)
	ooRatio := ratio(st.OutOfOrder, st.TotalEvents)
	dupRatio := ratio(st.DuplicateEvents, st.TotalEvents)
	missRatio := ratio(st.MissingEvents, st.TotalEvents)
	dq := marketmath.DQPenalty(staleness, lateRatio, ooRatio, dupRatio, missRatio)

	st.LastPrice = price
	st.PriceHist.Push(price)
	if eventAt.After(st.LastEvent) {
		st.LastEvent = eventAt
	}

	payload := map[string]float64{
		"ret_1s":              ret,
		"ret_1m":              ret1m,
		"ret_5m":              ret5m,
		"log_ret_1s":          logRet,
		"log_ret_1m":          logRet1m,
		"log_ret_5m":          logRet5m,
		"mean_ret_60":         meanRet60,
		"std_ret_60":          stdRet60,
		"z_ret_60":            z60,
		"mean_ret_300":        meanRet300,
		"std_ret_300":         stdRet300,
		"z_ret_300":           z300,
		"ewma_var_60":         st.EWMA60.Var,
		"ewma_var_300":        st.EWMA300.Var,
		"ewma_vol_60":         math.Sqrt(math.Max(st.EWMA60.Var, 1e-12)),
		"ewma_vol_300":        math.Sqrt(math.Max(st.EWMA300.Var, 1e-12)),
		"realized_var_300":    st.Log300.SumSq(),
		"realized_vol_300":    math.Sqrt(math.Max(st.Log300.SumSq(), 0)),
		"mean_vol_60":         meanVol60,
		"std_vol_60":          stdVol60,
		"volume_ratio_60":     vol / math.Max(meanVol60, 1e-9),
		"volume_surprise_60":  volSurprise,
		"anomaly_density_300": st.RecentAnomaly.Mean(),
		"tick_gap_s":          tickGap,
		"staleness_score":     staleness,
		"missingness_ratio":   missRatio,
		"out_of_order_ratio":  ooRatio,
		"duplicate_ratio":     dupRatio,
		"late_ratio":          lateRatio,
		"dq_penalty":          dq,
	}
	return payload, dq
}

func ratio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}

func horizonReturn(hist *marketmath.Ring, current float64, lag int) float64 {
	if hist == nil || lag < 1 {
		return 0
	}
	base, ok := hist.AtLag(lag - 1)
	if !ok || base <= 0 {
		return 0
	}
	return (current / base) - 1
}

func horizonLogReturn(hist *marketmath.Ring, current float64, lag int) float64 {
	if hist == nil || lag < 1 {
		return 0
	}
	base, ok := hist.AtLag(lag - 1)
	if !ok || base <= 0 {
		return 0
	}
	return math.Log(math.Max(current, eps) / math.Max(base, eps))
}
