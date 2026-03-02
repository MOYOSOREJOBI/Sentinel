package main

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"sentinel/internal/auth"
)

func TestRateLimitSubjectKey_PrioritizesAuthorization(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tok, err := auth.Sign("analyst@example.com", "analyst", priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	keyFn := rateLimitSubjectKey("gov", &priv.PublicKey)
	req := httptest.NewRequest(http.MethodPost, "/models/deploy", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer "+tok)
	req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: "cookie-token"})

	if got := keyFn(req); got != "gov:sub:analyst@example.com" {
		t.Fatalf("unexpected key: %s", got)
	}
}

func TestRateLimitSubjectKey_UsesCookieThenIP(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tok, err := auth.Sign("ops@example.com", "operator", priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	keyFn := rateLimitSubjectKey("workflow", &priv.PublicKey)
	req := httptest.NewRequest(http.MethodPost, "/replay/start", nil)
	req.RemoteAddr = "10.0.0.2:8080"
	req.AddCookie(&http.Cookie{Name: "sentinel_token", Value: tok})

	if got := keyFn(req); got != "workflow:sub:ops@example.com" {
		t.Fatalf("unexpected cookie key: %s", got)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/replay/start", nil)
	req2.RemoteAddr = "10.0.0.3:9090"
	if got := keyFn(req2); got != "workflow:ip:10.0.0.3:9090" {
		t.Fatalf("unexpected ip key: %s", got)
	}
}

func TestRateLimitSubjectKey_FallbackForInvalidBearerToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	keyFn := rateLimitSubjectKey("gov", &priv.PublicKey)
	req := httptest.NewRequest(http.MethodPost, "/models/deploy", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	if got := keyFn(req); got != "gov:authz:not-a-jwt" {
		t.Fatalf("unexpected fallback key: %s", got)
	}
}
