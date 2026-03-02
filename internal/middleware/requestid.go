package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDContextKey struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			rid = hex.EncodeToString(b)
		}
		r.Header.Set("X-Request-ID", rid)
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, rid)))
	})
}

func RequestIDValue(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Context().Value(requestIDContextKey{}); v != nil {
		if rid, ok := v.(string); ok {
			return rid
		}
	}
	return r.Header.Get("X-Request-ID")
}
