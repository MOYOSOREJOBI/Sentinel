package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"sentinel/internal/rediskv"
)

type entry struct {
	b   []byte
	exp time.Time
}

var mem sync.Map
var redisClient = rediskv.NewFromEnv()

func GetOrLoadJSON[T any](ctx context.Context, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var zero T
	if redisClient != nil && redisClient.Enabled() {
		if raw, ok, err := redisClient.Get(ctx, key); err == nil && ok {
			var out T
			if json.Unmarshal([]byte(raw), &out) == nil {
				return out, nil
			}
		}
	}
	if v, ok := mem.Load(key); ok {
		e := v.(entry)
		if time.Now().Before(e.exp) {
			var out T
			if json.Unmarshal(e.b, &out) == nil {
				return out, nil
			}
		}
	}
	out, err := loader()
	if err != nil {
		return zero, err
	}
	if b, err := json.Marshal(out); err == nil {
		mem.Store(key, entry{b: b, exp: time.Now().Add(ttl)})
		if redisClient != nil && redisClient.Enabled() {
			_ = redisClient.SetEX(ctx, key, string(b), ttl)
		}
	}
	return out, nil
}
