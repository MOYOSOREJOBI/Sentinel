package healthcheck

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestMustPort(t *testing.T) {
	t.Setenv("HC_PORT", "1234")
	if got := MustPort("HC_PORT", 9999); got != 1234 {
		t.Fatalf("MustPort()=%d want 1234", got)
	}

	t.Setenv("HC_PORT", "not-a-port")
	if got := MustPort("HC_PORT", 9999); got != 9999 {
		t.Fatalf("MustPort fallback=%d want 9999", got)
	}
}

func TestRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, p, err := net.SplitHostPort(ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	if code := Run(port, "/readyz"); code != 0 {
		t.Fatalf("Run()=%d want 0", code)
	}
	if code := Run(port, "/missing"); code == 0 {
		t.Fatalf("Run() for missing path=%d want non-zero", code)
	}
}
