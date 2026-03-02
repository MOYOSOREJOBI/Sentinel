package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"sentinel/internal/rediskv"
)

type limiterEntry struct {
	Count int
	Reset time.Time
}

var rl sync.Map
var rlRedis = rediskv.NewFromEnv()

func allow(key string, max int, window time.Duration) bool {
	if rlRedis != nil && rlRedis.Enabled() {
		ctx := context.Background()
		n, err := rlRedis.Incr(ctx, "rl:"+key)
		if err == nil {
			if n == 1 {
				_ = rlRedis.Expire(ctx, "rl:"+key, window)
			}
			return n <= max
		}
	}
	now := time.Now()
	v, _ := rl.LoadOrStore(key, limiterEntry{Count: 0, Reset: now.Add(window)})
	e := v.(limiterEntry)
	if now.After(e.Reset) {
		e = limiterEntry{Count: 0, Reset: now.Add(window)}
	}
	e.Count++
	rl.Store(key, e)
	return e.Count <= max
}

func RateLimit(keyFn func(*http.Request) string, max int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allow(keyFn(r), max, window) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "rate_limited", "limit": strconv.Itoa(max)})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
