package main

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"sentinel/internal/auth"
	"sentinel/internal/authz"
	"sentinel/internal/middleware"
)

func TestLogoutRequiresAuthAndCSRF(t *testing.T) {
	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		middleware.ReloadSecurityBackendsFromEnv()
		loginAttempts = sync.Map{}
	})
	_ = os.Setenv("REDIS_ADDR", "127.0.0.1:1")
	_ = os.Setenv("REDIS_ENABLED", "true")
	_ = os.Setenv("DEV_UNSAFE", "true")
	middleware.ReloadSecurityBackendsFromEnv()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	token, err := auth.Sign("admin@sentinel.local", "admin", priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if err := middleware.BindCSRF("admin@sentinel.local", "csrf-logout", time.Minute); err != nil {
		t.Fatalf("bind csrf: %v", err)
	}
	t.Cleanup(func() {
		middleware.ClearCSRF("admin@sentinel.local")
	})

	r := chi.NewRouter()
	r.Use(middleware.RequireCSRFFunc(authz.SubjectFromRequest(&priv.PublicKey)))
	r.With(authz.RequireAuth(&priv.PublicKey)).Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	unauth := httptest.NewRecorder()
	r.ServeHTTP(unauth, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth logout status=%d want %d", unauth.Code, http.StatusUnauthorized)
	}

	missingCSRFReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	missingCSRFReq.AddCookie(&http.Cookie{Name: "sentinel_token", Value: token})
	missingCSRF := httptest.NewRecorder()
	r.ServeHTTP(missingCSRF, missingCSRFReq)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status=%d want %d", missingCSRF.Code, http.StatusForbidden)
	}

	okReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	okReq.AddCookie(&http.Cookie{Name: "sentinel_token", Value: token})
	okReq.Header.Set("X-CSRF-Token", "csrf-logout")
	ok := httptest.NewRecorder()
	r.ServeHTTP(ok, okReq)
	if ok.Code != http.StatusNoContent {
		t.Fatalf("valid logout status=%d want %d", ok.Code, http.StatusNoContent)
	}
}
