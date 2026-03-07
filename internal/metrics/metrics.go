package metrics

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled by Sentinel Go services.",
		},
		[]string{"service", "method", "path", "status"},
	)
	httpLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"service", "path"},
	)
	dbLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query latency in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"service", "query"},
	)
	queryRowsReturned = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "query_rows_returned",
			Help:    "Rows returned by bounded query endpoints.",
			Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 200, 250, 500},
		},
		[]string{"service", "endpoint"},
	)
	caseMutations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "case_mutations_total",
			Help: "Case workflow mutations handled by Sentinel services.",
		},
		[]string{"service", "action"},
	)
	replayJobEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "replay_job_events_total",
			Help: "Replay job lifecycle events emitted by Sentinel services.",
		},
		[]string{"service", "status"},
	)
	kafkaProduced = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Kafka messages produced by Go services.",
		},
		[]string{"service", "topic"},
	)
	kafkaConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Kafka messages consumed by Go services.",
		},
		[]string{"service", "topic"},
	)
	serviceUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_up",
			Help: "Whether the service process is up.",
		},
		[]string{"service"},
	)

	metricsHandler = promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	)
)

func init() {
	registry.MustRegister(httpRequests, httpLatency, dbLatency, queryRowsReturned, caseMutations, replayJobEvents, kafkaProduced, kafkaConsumed, serviceUp)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Flush() {
	if flusher, ok := s.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := s.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (s *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := s.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (s *statusRecorder) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := s.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(s.ResponseWriter, r)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func Handler(w http.ResponseWriter, r *http.Request) {
	metricsHandler.ServeHTTP(w, r)
}

func HTTPMiddleware(service string) func(http.Handler) http.Handler {
	serviceUp.WithLabelValues(service).Set(1)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			path := routePattern(r)
			httpRequests.WithLabelValues(service, r.Method, path, strconv.Itoa(rec.status)).Inc()
			httpLatency.WithLabelValues(service, path).Observe(time.Since(start).Seconds())
		})
	}
}

func Wrap(service string, next http.Handler) http.Handler {
	return HTTPMiddleware(service)(next)
}

func IncKafkaProduced(service, topic string) {
	kafkaProduced.WithLabelValues(service, topic).Inc()
}

func IncKafkaConsumed(service, topic string) {
	kafkaConsumed.WithLabelValues(service, topic).Inc()
}

func ObserveDBQuery(service, query string, d time.Duration) {
	dbLatency.WithLabelValues(service, sanitizeQuery(query)).Observe(d.Seconds())
}

func ObserveRowsReturned(service, endpoint string, rows int) {
	if rows < 0 {
		rows = 0
	}
	queryRowsReturned.WithLabelValues(service, sanitizeQuery(endpoint)).Observe(float64(rows))
}

func IncCaseMutation(service, action string) {
	caseMutations.WithLabelValues(service, sanitizeQuery(action)).Inc()
}

func IncReplayJobEvent(service, status string) {
	replayJobEvents.WithLabelValues(service, sanitizeQuery(status)).Inc()
}

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := strings.TrimSpace(rc.RoutePattern()); p != "" {
			return p
		}
	}
	if p := strings.TrimSpace(r.URL.Path); p != "" {
		return p
	}
	return "/"
}

func sanitizeQuery(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	if in == "" {
		return "unknown"
	}
	in = strings.Join(strings.Fields(in), "_")
	if len(in) > 80 {
		in = in[:80]
	}
	return in
}
