package middleware

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
	"sentinel/internal/rediskv"
)

func TestQueryStyleWriteRequiresCSRFAgainstAuthenticatedAdmin(t *testing.T) {
	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		csrfRedis = rediskv.NewFromEnv()
		csrfStore = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", "127.0.0.1:1")
	_ = os.Setenv("REDIS_ENABLED", "true")
	_ = os.Setenv("DEV_UNSAFE", "true")
	csrfRedis = rediskv.NewFromEnv()
	csrfStore = sync.Map{}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	adminToken, err := auth.Sign("admin@sentinel.local", "admin", priv)
	if err != nil {
		t.Fatalf("sign admin: %v", err)
	}
	viewerToken, err := auth.Sign("viewer@sentinel.local", "viewer", priv)
	if err != nil {
		t.Fatalf("sign viewer: %v", err)
	}

	r := chi.NewRouter()
	r.Use(RequireCSRFFunc(authz.SubjectFromRequest(&priv.PublicKey)))
	r.With(authz.RequirePerm(&priv.PublicKey, "admin")).Post("/dev/backfill", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	missingCSRFReq := httptest.NewRequest(http.MethodPost, "/dev/backfill", nil)
	missingCSRFReq.AddCookie(&http.Cookie{Name: "sentinel_token", Value: adminToken})
	missingCSRF := httptest.NewRecorder()
	r.ServeHTTP(missingCSRF, missingCSRFReq)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status=%d want %d", missingCSRF.Code, http.StatusForbidden)
	}

	if err := BindCSRF("admin@sentinel.local", "csrf-admin", time.Minute); err != nil {
		t.Fatalf("bind admin csrf: %v", err)
	}
	adminReq := httptest.NewRequest(http.MethodPost, "/dev/backfill", nil)
	adminReq.AddCookie(&http.Cookie{Name: "sentinel_token", Value: adminToken})
	adminReq.Header.Set("X-CSRF-Token", "csrf-admin")
	admin := httptest.NewRecorder()
	r.ServeHTTP(admin, adminReq)
	if admin.Code != http.StatusNoContent {
		t.Fatalf("admin status=%d want %d", admin.Code, http.StatusNoContent)
	}

	if err := BindCSRF("viewer@sentinel.local", "csrf-viewer", time.Minute); err != nil {
		t.Fatalf("bind viewer csrf: %v", err)
	}
	viewerReq := httptest.NewRequest(http.MethodPost, "/dev/backfill", nil)
	viewerReq.AddCookie(&http.Cookie{Name: "sentinel_token", Value: viewerToken})
	viewerReq.Header.Set("X-CSRF-Token", "csrf-viewer")
	viewer := httptest.NewRecorder()
	r.ServeHTTP(viewer, viewerReq)
	if viewer.Code != http.StatusForbidden {
		t.Fatalf("viewer status=%d want %d", viewer.Code, http.StatusForbidden)
	}
}

func TestProtectedWriteFailsClosedWhenRedisUnavailableAndUnsafeDisabled(t *testing.T) {
	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		csrfRedis = rediskv.NewFromEnv()
		csrfStore = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", "127.0.0.1:1")
	_ = os.Setenv("REDIS_ENABLED", "true")
	_ = os.Setenv("DEV_UNSAFE", "false")
	csrfRedis = rediskv.NewFromEnv()
	csrfStore = sync.Map{}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	adminToken, err := auth.Sign("admin@sentinel.local", "admin", priv)
	if err != nil {
		t.Fatalf("sign admin: %v", err)
	}

	r := chi.NewRouter()
	r.Use(RequireCSRFFunc(authz.SubjectFromRequest(&priv.PublicKey)))
	r.With(authz.RequirePerm(&priv.PublicKey, "admin")).Post("/dev/backfill", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/dev/backfill", nil)
	req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: adminToken})
	req.Header.Set("X-CSRF-Token", "present-but-unverifiable")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusServiceUnavailable)
	}
}
