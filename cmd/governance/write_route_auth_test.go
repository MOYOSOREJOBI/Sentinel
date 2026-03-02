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

func TestGovernanceWriteRoutesEnforceAuthAndRBAC(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	router := chi.NewRouter()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router.With(authz.RequireAnyPerm(&priv.PublicKey, "*", "model:deploy")).Post("/models/deploy", ok)
	router.With(authz.RequireAnyPerm(&priv.PublicKey, "replay:write", "*")).Post("/replay/start", ok)
	router.With(authz.RequireAnyPerm(&priv.PublicKey, "*")).Put("/escalation-policies", ok)

	adminToken, err := auth.Sign("admin@sentinel.local", "admin", priv)
	if err != nil {
		t.Fatalf("sign admin: %v", err)
	}
	analystToken, err := auth.Sign("analyst@sentinel.local", "analyst", priv)
	if err != nil {
		t.Fatalf("sign analyst: %v", err)
	}
	viewerToken, err := auth.Sign("viewer@sentinel.local", "viewer", priv)
	if err != nil {
		t.Fatalf("sign viewer: %v", err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		token  string
		want   int
	}{
		{name: "deploy unauth", method: http.MethodPost, path: "/models/deploy", want: http.StatusUnauthorized},
		{name: "deploy analyst forbidden", method: http.MethodPost, path: "/models/deploy", token: analystToken, want: http.StatusForbidden},
		{name: "deploy admin ok", method: http.MethodPost, path: "/models/deploy", token: adminToken, want: http.StatusNoContent},
		{name: "replay viewer forbidden", method: http.MethodPost, path: "/replay/start", token: viewerToken, want: http.StatusForbidden},
		{name: "replay analyst ok", method: http.MethodPost, path: "/replay/start", token: analystToken, want: http.StatusNoContent},
		{name: "policy viewer forbidden", method: http.MethodPut, path: "/escalation-policies", token: viewerToken, want: http.StatusForbidden},
		{name: "policy admin ok", method: http.MethodPut, path: "/escalation-policies", token: adminToken, want: http.StatusNoContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: tc.token})
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d want %d", rr.Code, tc.want)
			}
		})
	}
}
