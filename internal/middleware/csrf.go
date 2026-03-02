package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"sentinel/internal/rediskv"
)

type csrfEntry struct {
	Token  string
	Expiry time.Time
}

var csrfStore sync.Map
var csrfRedis = rediskv.NewFromEnv()

func NewCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func BindCSRF(subject, token string, ttl time.Duration) {
	csrfStore.Store(subject, csrfEntry{Token: token, Expiry: time.Now().Add(ttl)})
	if csrfRedis != nil && csrfRedis.Enabled() {
		_ = csrfRedis.SetEX(context.Background(), "csrf:"+subject, token, ttl)
	}
}

func ClearCSRF(subject string) {
	csrfStore.Delete(subject)
	if csrfRedis != nil && csrfRedis.Enabled() {
		_ = csrfRedis.Del(context.Background(), "csrf:"+subject)
	}
}

func ValidCSRF(subject, token string) bool {
	if csrfRedis != nil && csrfRedis.Enabled() {
		if v, ok, err := csrfRedis.Get(context.Background(), "csrf:"+subject); err == nil && ok {
			return token != "" && v == token
		}
	}
	v, ok := csrfStore.Load(subject)
	if !ok {
		return false
	}
	e := v.(csrfEntry)
	if time.Now().After(e.Expiry) || token == "" {
		return false
	}
	return e.Token == token
}

func RequireCSRFFunc(subjectFromReq func(*http.Request) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			// Login is the CSRF bootstrap step that issues the CSRF cookie/token pair.
			if r.URL.Path == "/auth/login" {
				next.ServeHTTP(w, r)
				return
			}
			sub, ok := subjectFromReq(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if !ValidCSRF(sub, r.Header.Get("X-CSRF-Token")) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
