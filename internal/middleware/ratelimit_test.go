package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"sentinel/internal/rediskv"
)

func TestRateLimit429(t *testing.T) {
	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		rlRedis = rediskv.NewFromEnv()
		rl = sync.Map{}
	})
	_ = os.Setenv("REDIS_ADDR", "127.0.0.1:1")
	_ = os.Setenv("REDIS_ENABLED", "true")
	_ = os.Setenv("DEV_UNSAFE", "true")
	rlRedis = rediskv.NewFromEnv()
	rl = sync.Map{}

	h := RateLimit(func(r *http.Request) string { return "k" }, 1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, httptest.NewRequest("GET", "/", nil))
	if rr2.Code != 429 {
		t.Fatalf("expected 429")
	}
}
