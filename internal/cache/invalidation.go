package cache

import (
	"context"
	"strings"
)

func InvalidateByPrefixes(ctx context.Context, prefixes ...string) {
	mem.Range(func(key, _ any) bool {
		ks, ok := key.(string)
		if !ok {
			return true
		}
		for _, p := range prefixes {
			if strings.HasPrefix(ks, p) {
				mem.Delete(ks)
				break
			}
		}
		return true
	})
	if redisClient != nil && redisClient.Enabled() {
		for _, p := range prefixes {
			_ = redisClient.Del(ctx, p+"*")
		}
	}
}
