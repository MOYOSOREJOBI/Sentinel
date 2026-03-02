package main

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"sentinel/internal/auth"
	"sentinel/internal/authz"
)

func TestAlertAckAuthz(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.With(authz.RequireAnyPerm(&priv.PublicKey, "alerts:write", "*")).Post("/alerts/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("unauthenticated is 401", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/alerts/1/ack", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("viewer is 403", func(t *testing.T) {
		token, err := auth.Sign("viewer@sentinel.local", "viewer", priv)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/alerts/1/ack", nil)
		req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: token})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status=%d want %d", rr.Code, http.StatusForbidden)
		}
	})
}
