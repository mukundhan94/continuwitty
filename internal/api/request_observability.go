package api

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"engram/internal/chat"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RequestMetricsRecorder captures API request telemetry samples.
type RequestMetricsRecorder interface {
	Record(sample RequestMetricSample)
	Snapshot() RequestMetricsSnapshot
}

// RequestMetricSample is one observed request/response sample.
type RequestMetricSample struct {
	Domain     string
	Method     string
	Route      string
	StatusCode int
	Duration   time.Duration
}

// RequestMetricsSnapshot is a stable JSON payload for operators.
type RequestMetricsSnapshot struct {
	Totals        RequestTotals        `json:"totals"`
	ByStatusClass map[string]int64     `json:"by_status_class"`
	ByDomain      map[string]int64     `json:"by_domain"`
	ByRoute       map[string]RouteHit  `json:"by_route"`
	ProviderFails map[string]int64     `json:"provider_failures"`
	StreamHealth  map[string]StreamHit `json:"stream_health"`
	Lifecycle     map[string]int64     `json:"lifecycle_traces"`
}

// RequestTotals contains top-level request counters.
type RequestTotals struct {
	Requests int64 `json:"requests"`
	Errors   int64 `json:"errors"`
}

// RouteHit aggregates route-level counters and latency.
type RouteHit struct {
	Count               int64   `json:"count"`
	ErrorCount          int64   `json:"error_count"`
	TotalDurationMillis float64 `json:"total_duration_ms"`
	AverageDurationMS   float64 `json:"avg_duration_ms"`
}

// StreamHit aggregates stream-health counters and latency by stream category.
type StreamHit struct {
	Count               int64   `json:"count"`
	ErrorCount          int64   `json:"error_count"`
	TotalChunks         int64   `json:"total_chunks"`
	TotalDurationMillis float64 `json:"total_duration_ms"`
	AverageDurationMS   float64 `json:"avg_duration_ms"`
}

// InMemoryRequestMetrics is a process-local request metrics recorder.
type InMemoryRequestMetrics struct {
	mutex         sync.RWMutex
	totalRequests int64
	totalErrors   int64
	byStatusClass map[string]int64
	byDomain      map[string]int64
	byRoute       map[string]routeAccumulator
	providerFails map[string]int64
	streamHealth  map[string]streamAccumulator
	lifecycle     map[string]int64
}

type routeAccumulator struct {
	count              int64
	errorCount         int64
	totalDurationMilli float64
}

type streamAccumulator struct {
	count              int64
	errorCount         int64
	totalChunks        int64
	totalDurationMilli float64
}

// NewInMemoryRequestMetrics initializes a process-local metrics recorder.
func NewInMemoryRequestMetrics() *InMemoryRequestMetrics {
	return &InMemoryRequestMetrics{
		byStatusClass: make(map[string]int64),
		byDomain:      make(map[string]int64),
		byRoute:       make(map[string]routeAccumulator),
		providerFails: make(map[string]int64),
		streamHealth:  make(map[string]streamAccumulator),
		lifecycle:     make(map[string]int64),
	}
}

// Record appends one telemetry sample.
func (metrics *InMemoryRequestMetrics) Record(sample RequestMetricSample) {
	if metrics == nil {
		return
	}
	domain := normalizedMetricValue(sample.Domain, "unknown")
	method := normalizedMetricValue(sample.Method, "UNKNOWN")
	route := normalizedMetricValue(sample.Route, "unknown")
	statusClass := statusCodeClass(sample.StatusCode)
	durationMillis := durationToMillis(sample.Duration)
	routeKey := method + " " + route

	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	metrics.totalRequests++
	if sample.StatusCode >= http.StatusInternalServerError {
		metrics.totalErrors++
	}
	metrics.byStatusClass[statusClass]++
	metrics.byDomain[domain]++

	accumulator := metrics.byRoute[routeKey]
	accumulator.count++
	if sample.StatusCode >= http.StatusInternalServerError {
		accumulator.errorCount++
	}
	accumulator.totalDurationMilli += durationMillis
	metrics.byRoute[routeKey] = accumulator
}

// Snapshot returns a copy safe for JSON serialization.
func (metrics *InMemoryRequestMetrics) Snapshot() RequestMetricsSnapshot {
	if metrics == nil {
		return RequestMetricsSnapshot{
			ByStatusClass: map[string]int64{},
			ByDomain:      map[string]int64{},
			ByRoute:       map[string]RouteHit{},
			ProviderFails: map[string]int64{},
			StreamHealth:  map[string]StreamHit{},
			Lifecycle:     map[string]int64{},
		}
	}
	metrics.mutex.RLock()
	defer metrics.mutex.RUnlock()

	snapshot := RequestMetricsSnapshot{
		Totals: RequestTotals{
			Requests: metrics.totalRequests,
			Errors:   metrics.totalErrors,
		},
		ByStatusClass: cloneInt64Map(metrics.byStatusClass),
		ByDomain:      cloneInt64Map(metrics.byDomain),
		ByRoute:       make(map[string]RouteHit, len(metrics.byRoute)),
		ProviderFails: cloneInt64Map(metrics.providerFails),
		StreamHealth:  make(map[string]StreamHit, len(metrics.streamHealth)),
		Lifecycle:     cloneInt64Map(metrics.lifecycle),
	}
	for routeKey, accumulator := range metrics.byRoute {
		average := 0.0
		if accumulator.count > 0 {
			average = accumulator.totalDurationMilli / float64(accumulator.count)
		}
		snapshot.ByRoute[routeKey] = RouteHit{
			Count:               accumulator.count,
			ErrorCount:          accumulator.errorCount,
			TotalDurationMillis: accumulator.totalDurationMilli,
			AverageDurationMS:   average,
		}
	}
	for key, accumulator := range metrics.streamHealth {
		average := 0.0
		if accumulator.count > 0 {
			average = accumulator.totalDurationMilli / float64(accumulator.count)
		}
		snapshot.StreamHealth[key] = StreamHit{
			Count:               accumulator.count,
			ErrorCount:          accumulator.errorCount,
			TotalChunks:         accumulator.totalChunks,
			TotalDurationMillis: accumulator.totalDurationMilli,
			AverageDurationMS:   average,
		}
	}
	return snapshot
}

// RecordProviderFailure captures a categorized provider failure sample from chat service flow.
func (metrics *InMemoryRequestMetrics) RecordProviderFailure(sample chat.ProviderFailureSample) {
	if metrics == nil {
		return
	}
	key := providerFailureKey(sample)
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	metrics.providerFails[key]++
}

// RecordStreamHealth captures stream outcome telemetry from chat service flow.
func (metrics *InMemoryRequestMetrics) RecordStreamHealth(sample chat.StreamHealthSample) {
	if metrics == nil {
		return
	}
	key := streamHealthKey(sample)
	durationMillis := durationToMillis(sample.Duration)

	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	accumulator := metrics.streamHealth[key]
	accumulator.count++
	if sample.Outcome != "completed" {
		accumulator.errorCount++
	}
	accumulator.totalChunks += int64(max(sample.ChunkCount, 0))
	accumulator.totalDurationMilli += durationMillis
	metrics.streamHealth[key] = accumulator
}

// RecordLifecycleTrace captures lifecycle stage transitions from chat send/stream flows.
func (metrics *InMemoryRequestMetrics) RecordLifecycleTrace(sample chat.LifecycleTraceSample) {
	if metrics == nil {
		return
	}
	key := lifecycleTraceKey(sample)
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	metrics.lifecycle[key]++
}

func cloneInt64Map(source map[string]int64) map[string]int64 {
	cloned := make(map[string]int64, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func statusCodeClass(statusCode int) string {
	if statusCode <= 0 {
		return "unknown"
	}
	return string([]byte{byte('0' + statusCode/100), 'x', 'x'})
}

func durationToMillis(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000.0
}

func normalizedMetricValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func providerFailureKey(sample chat.ProviderFailureSample) string {
	operation := normalizedMetricValue(sample.Operation, "unknown")
	provider := normalizedMetricValue(string(sample.Provider), "unknown")
	errorCode := normalizedMetricValue(sample.ErrorCode, "provider_error")
	return operation + " " + provider + " " + errorCode
}

func streamHealthKey(sample chat.StreamHealthSample) string {
	operation := normalizedMetricValue(sample.Operation, "unknown")
	provider := normalizedMetricValue(string(sample.Provider), "unknown")
	outcome := normalizedMetricValue(sample.Outcome, "unknown")
	return operation + " " + provider + " " + outcome
}

func lifecycleTraceKey(sample chat.LifecycleTraceSample) string {
	operation := normalizedMetricValue(sample.Operation, "unknown")
	stage := normalizedMetricValue(sample.Stage, "unknown")
	errorCode := normalizedMetricValue(sample.ErrorCode, "")
	if errorCode == "" {
		return operation + " " + stage
	}
	return operation + " " + stage + " " + errorCode
}

func requestDomain(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/mcp"):
		return "mcp"
	case strings.HasPrefix(path, "/api/v1"):
		return "api"
	case strings.HasPrefix(path, "/oauth"), strings.HasPrefix(path, "/.well-known"):
		return "oauth"
	case strings.HasPrefix(path, "/ui"), path == "/", strings.HasPrefix(path, "/login"), strings.HasPrefix(path, "/logout"):
		return "ui"
	default:
		return "other"
	}
}

func requestRoutePattern(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	routeContext := chi.RouteContext(request.Context())
	if routeContext == nil {
		return request.URL.Path
	}
	pattern := strings.TrimSpace(routeContext.RoutePattern())
	if pattern == "" {
		return request.URL.Path
	}
	return pattern
}

func requestIDFromContext(request *http.Request) string {
	if request == nil {
		return ""
	}
	return middleware.GetReqID(request.Context())
}

type responseStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (recorder *responseStatusRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *responseStatusRecorder) status() int {
	if recorder.statusCode > 0 {
		return recorder.statusCode
	}
	return http.StatusOK
}

func newRequestTelemetryMiddleware(
	logger *slog.Logger,
	metricsRecorder RequestMetricsRecorder,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			start := time.Now()
			recorder := &responseStatusRecorder{ResponseWriter: writer}
			next.ServeHTTP(recorder, request)

			route := requestRoutePattern(request)
			domain := requestDomain(request.URL.Path)
			statusCode := recorder.status()
			duration := time.Since(start)
			if metricsRecorder != nil {
				metricsRecorder.Record(RequestMetricSample{
					Domain:     domain,
					Method:     request.Method,
					Route:      route,
					StatusCode: statusCode,
					Duration:   duration,
				})
			}
			if logger == nil {
				return
			}
			logger.Info(
				"http_request",
				"request_id", requestIDFromContext(request),
				"domain", domain,
				"method", request.Method,
				"route", route,
				"path", request.URL.Path,
				"status", statusCode,
				"duration_ms", durationToMillis(duration),
				"remote_ip", request.RemoteAddr,
			)
		})
	}
}
