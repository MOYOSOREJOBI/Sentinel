package cache

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"sentinel/internal/rediskv"
	"sentinel/internal/testredis"
)

func TestGetOrLoadJSON_RedisTTLExpiry(t *testing.T) {
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
		redisClient = rediskv.NewFromEnv()
		mem = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", srv.Addr())
	_ = os.Setenv("REDIS_ENABLED", "true")
	redisClient = rediskv.NewFromEnv()
	mem = sync.Map{}

	calls := 0
	loader := func() (map[string]int, error) {
		calls++
		return map[string]int{"v": calls}, nil
	}

	ctx := context.Background()
	out1, _ := GetOrLoadJSON(ctx, "cache:k", 100*time.Millisecond, loader)
	out2, _ := GetOrLoadJSON(ctx, "cache:k", 100*time.Millisecond, loader)
	if out1["v"] != 1 || out2["v"] != 1 || calls != 1 {
		t.Fatalf("expected cache hit before ttl expiry, calls=%d out1=%v out2=%v", calls, out1, out2)
	}

	time.Sleep(160 * time.Millisecond)
	mem = sync.Map{}
	out3, _ := GetOrLoadJSON(ctx, "cache:k", 100*time.Millisecond, loader)
	if out3["v"] != 2 || calls != 2 {
		t.Fatalf("expected cache miss after ttl expiry, calls=%d out3=%v", calls, out3)
	}
}
