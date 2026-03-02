package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireForbidden(t *testing.T) {
	h := Require("viewer", "alerts:write", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/alerts/1/ack", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
