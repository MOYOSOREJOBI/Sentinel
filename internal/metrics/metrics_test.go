package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerExposesPrometheusMetrics(t *testing.T) {
	IncKafkaProduced("alerts", "alerts.created")
	ObserveDBQuery("query", "SELECT * FROM incidents", 10*time.Millisecond)
	IncCaseMutation("alerts", "create")
	IncReplayJobEvent("governance", "queued")

	handler := HTTPMiddleware("query")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	rrReq := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/queue", nil)
	handler.ServeHTTP(rrReq, req)

	rr := httptest.NewRecorder()
	Handler(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"http_requests_total{method=\"GET\",path=\"/queue\",service=\"query\",status=\"201\"}",
		"http_request_duration_seconds_bucket",
		"db_query_duration_seconds_bucket",
		"kafka_messages_produced_total{service=\"alerts\",topic=\"alerts.created\"}",
		"case_mutations_total{action=\"create\",service=\"alerts\"}",
		"replay_job_events_total{service=\"governance\",status=\"queued\"}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in metrics body:\n%s", want, body)
		}
	}
}
