package authz

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"sentinel/internal/auth"
)

func TestRequirePermReturns401WhenUnauthenticated(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	h := RequirePerm(&priv.PublicKey, "read")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/secure", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequirePermReturns403WhenRoleLacksPermission(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.Sign("viewer@sentinel.local", "viewer", priv)
	if err != nil {
		t.Fatal(err)
	}
	h := RequirePerm(&priv.PublicKey, "alerts:write")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusForbidden)
	}
}
