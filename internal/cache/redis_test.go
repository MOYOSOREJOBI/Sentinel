package cache

import (
	"context"
	"testing"
	"time"
)

func TestGetOrLoadJSON(t *testing.T) {
	n := 0
	loader := func() (map[string]int, error) { n++; return map[string]int{"x": 1}, nil }
	_, _ = GetOrLoadJSON(context.Background(), "k", time.Minute, loader)
	_, _ = GetOrLoadJSON(context.Background(), "k", time.Minute, loader)
	if n != 1 {
		t.Fatalf("expected cache hit")
	}
}
