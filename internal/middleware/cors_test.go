package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORSAllowlist(t *testing.T) {
	_ = os.Setenv("ALLOWED_ORIGINS", "http://example.com")
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://bad.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatalf("expected 403")
	}
}
