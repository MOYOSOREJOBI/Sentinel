package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
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
var errCSRFStoreUnavailable = errors.New("csrf store unavailable")

func NewCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func BindCSRF(subject, token string, ttl time.Duration) error {
	if csrfRedis != nil && csrfRedis.Enabled() {
		if err := csrfRedis.SetEX(context.Background(), "csrf:"+subject, token, ttl); err == nil {
			return nil
		} else {
			logSecurityStoreOnce("csrf:redis-bind", "csrf redis store unavailable; protected writes will fail closed until redis is healthy")
		}
	}
	if !allowUnsafeInMemorySecurity() {
		logSecurityStoreOnce("csrf:deny-bind", "csrf denied because redis is unavailable; set DEV_UNSAFE=true only for local development")
		return errCSRFStoreUnavailable
	}
	logSecurityStoreOnce("csrf:unsafe-bind", "csrf redis store unavailable; using in-memory fallback because DEV_UNSAFE is enabled")
	csrfStore.Store(subject, csrfEntry{Token: token, Expiry: time.Now().Add(ttl)})
	return nil
}

func ClearCSRF(subject string) {
	if csrfRedis != nil && csrfRedis.Enabled() {
		if err := csrfRedis.Del(context.Background(), "csrf:"+subject); err == nil {
			return
		}
		logSecurityStoreOnce("csrf:redis-clear", "csrf redis store unavailable during logout; clearing local cookies and any unsafe in-memory fallback only")
	}
	if allowUnsafeInMemorySecurity() {
		csrfStore.Delete(subject)
	}
}

func ValidCSRF(subject, token string) (bool, error) {
	if csrfRedis != nil && csrfRedis.Enabled() {
		if v, ok, err := csrfRedis.Get(context.Background(), "csrf:"+subject); err == nil && ok {
			return v == token, nil
		} else if err == nil && !ok {
			return false, nil
		}
		logSecurityStoreOnce("csrf:redis-read", "csrf redis store unavailable; protected writes will fail closed until redis is healthy")
	}
	if !allowUnsafeInMemorySecurity() {
		logSecurityStoreOnce("csrf:deny-read", "csrf validation denied because redis is unavailable; set DEV_UNSAFE=true only for local development")
		return false, errCSRFStoreUnavailable
	}
	logSecurityStoreOnce("csrf:unsafe-read", "csrf redis store unavailable; using in-memory fallback because DEV_UNSAFE is enabled")
	v, ok := csrfStore.Load(subject)
	if !ok {
		return false, nil
	}
	e := v.(csrfEntry)
	if time.Now().After(e.Expiry) {
		csrfStore.Delete(subject)
		return false, nil
	}
	return e.Token == token, nil
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
			token := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if token == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			valid, err := ValidCSRF(sub, token)
			if err != nil {
				http.Error(w, "security store unavailable", http.StatusServiceUnavailable)
				return
			}
			if !valid {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
