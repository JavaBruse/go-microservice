package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	ActiveRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_active",
			Help: "Number of active HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	AnomaliesDetected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "anomalies_detected_total",
			Help: "Total number of anomalies detected",
		},
	)

	MetricsProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "metrics_processed_total",
			Help: "Total number of metrics processed",
		},
	)

	RollingAverageRPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rolling_average_rps",
			Help: "Rolling average RPS (50 events window)",
		},
	)

	ProcessingQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "processing_queue_size",
			Help: "Current size of metrics processing queue",
		},
	)
)

func init() {
	prometheus.MustRegister(TotalRequests)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveRequests)
	prometheus.MustRegister(AnomaliesDetected)
	prometheus.MustRegister(MetricsProcessed)
	prometheus.MustRegister(RollingAverageRPS)
	prometheus.MustRegister(ProcessingQueueSize)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ActiveRequests.WithLabelValues(r.Method, r.URL.Path).Inc()

		rw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(rw, r)

		ActiveRequests.WithLabelValues(r.Method, r.URL.Path).Dec()

		duration := time.Since(start).Seconds()
		RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		TotalRequests.WithLabelValues(r.Method, r.URL.Path, http.StatusText(rw.status)).Inc()
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Функции для обновления метрик из сервиса аналитики
func IncrementMetricsProcessed() {
	MetricsProcessed.Inc()
}

func IncrementAnomaliesDetected() {
	AnomaliesDetected.Inc()
}

func SetRollingAverageRPS(value float64) {
	RollingAverageRPS.Set(value)
}

func SetProcessingQueueSize(value float64) {
	ProcessingQueueSize.Set(value)
}
