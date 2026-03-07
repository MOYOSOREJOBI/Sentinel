package middleware

import (
	"context"
	"encoding/json"
	"errors"
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

var errRateLimitStoreUnavailable = errors.New("rate limit store unavailable")

func allow(key string, max int, window time.Duration) (bool, error) {
	if rlRedis != nil && rlRedis.Enabled() {
		ctx := context.Background()
		n, err := rlRedis.Incr(ctx, "rl:"+key)
		if err == nil {
			if n == 1 {
				_ = rlRedis.Expire(ctx, "rl:"+key, window)
			}
			return n <= max, nil
		}
		logSecurityStoreOnce("ratelimit:redis", "rate limit redis store unavailable; protected writes will fail closed until redis is healthy")
	}
	if !allowUnsafeInMemorySecurity() {
		logSecurityStoreOnce("ratelimit:deny", "rate limiting denied because redis is unavailable; set DEV_UNSAFE=true only for local development")
		return false, errRateLimitStoreUnavailable
	}
	logSecurityStoreOnce("ratelimit:unsafe", "rate limit redis store unavailable; using in-memory fallback because DEV_UNSAFE is enabled")
	now := time.Now()
	v, _ := rl.LoadOrStore(key, limiterEntry{Count: 0, Reset: now.Add(window)})
	e := v.(limiterEntry)
	if now.After(e.Reset) {
		e = limiterEntry{Count: 0, Reset: now.Add(window)}
	}
	e.Count++
	rl.Store(key, e)
	return e.Count <= max, nil
}

func RateLimit(keyFn func(*http.Request) string, max int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := allow(keyFn(r), max, window)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "security_store_unavailable"})
				return
			}
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "rate_limited", "limit": strconv.Itoa(max)})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
