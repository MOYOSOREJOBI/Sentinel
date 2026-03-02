package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"sentinel/internal/rediskv"
	"sentinel/internal/testredis"
)

func TestCSRF_RedisBackedExpiry(t *testing.T) {
	srv, err := testredis.Start()
	if err != nil {
		t.Fatalf("start redis stub: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		csrfRedis = rediskv.NewFromEnv()
		csrfStore = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", srv.Addr())
	_ = os.Setenv("REDIS_ENABLED", "true")
	csrfRedis = rediskv.NewFromEnv()
	csrfStore = sync.Map{}

	BindCSRF("alice", "token-1", 100*time.Millisecond)
	if !ValidCSRF("alice", "token-1") {
		t.Fatalf("expected csrf token to validate before expiry")
	}
	time.Sleep(160 * time.Millisecond)
	if ValidCSRF("alice", "token-1") {
		t.Fatalf("expected csrf token to expire in redis")
	}
}

func TestRateLimit_RedisBackedBlocking(t *testing.T) {
	srv, err := testredis.Start()
	if err != nil {
		t.Fatalf("start redis stub: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		rlRedis = rediskv.NewFromEnv()
		rl = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", srv.Addr())
	_ = os.Setenv("REDIS_ENABLED", "true")
	rlRedis = rediskv.NewFromEnv()
	rl = sync.Map{}

	h := RateLimit(func(r *http.Request) string { return "redis-user" }, 2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected allowed request, got %d", rr.Code)
		}
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected redis-backed limiter to return 429, got %d", rr.Code)
	}
}
