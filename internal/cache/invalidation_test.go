package cache

import (
	"context"
	"testing"
	"time"
)

func TestInvalidateByPrefixes(t *testing.T) {
	_, _ = GetOrLoadJSON(context.Background(), "queue:v1:a", time.Minute, func() (map[string]int, error) { return map[string]int{"x": 1}, nil })
	InvalidateByPrefixes(context.Background(), "queue:v1:")
	count := 0
	_, _ = GetOrLoadJSON(context.Background(), "queue:v1:a", time.Minute, func() (map[string]int, error) { count++; return map[string]int{"x": 2}, nil })
	if count != 1 {
		t.Fatal("expected invalidated")
	}
}
