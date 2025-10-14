package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
)

// APIMetrics tracks API-related metrics
type APIMetrics struct {
	requestsTotal    *metrics.CounterVec
	requestDuration  *metrics.HistogramVec
	watchersActive   *metrics.GaugeVec
	validationErrors *metrics.CounterVec
	requestsInFlight *metrics.GaugeVec
	requestSize      *metrics.HistogramVec
	responseSize     *metrics.HistogramVec
}

// NewAPIMetrics creates API metrics collectors
func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{
		requestsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "requests_total",
				Help:      "Total number of API requests",
			},
			[]string{"resource", "verb", "status_code"},
		),

		requestDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "request_duration_seconds",
				Help:      "Duration of API requests",
				Buckets:   metrics.DefBuckets,
			},
			[]string{"resource", "verb"},
		),

		watchersActive: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "watchers_active",
				Help:      "Number of active watchers",
			},
			[]string{"resource"},
		),

		validationErrors: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "validation_errors_total",
				Help:      "Total number of validation errors",
			},
			[]string{"resource", "operation"},
		),

		requestsInFlight: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "requests_inflight",
				Help:      "Number of API requests currently being processed",
			},
			[]string{"resource", "verb"},
		),

		requestSize: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "request_size_bytes",
				Help:      "Size of API request payloads in bytes",
				Buckets:   metrics.ExponentialBuckets(100, 10, 8), // 100B to ~10MB
			},
			[]string{"resource", "verb"},
		),

		responseSize: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemAPI,
				Name:      "response_size_bytes",
				Help:      "Size of API response payloads in bytes",
				Buckets:   metrics.ExponentialBuckets(100, 10, 8), // 100B to ~10MB
			},
			[]string{"resource", "verb"},
		),
	}
}

// Register registers all API metrics
func (a *APIMetrics) Register(registry metrics.KubeRegistry) error {
	collectors := []metrics.Registerable{
		a.requestsTotal,
		a.requestDuration,
		a.watchersActive,
		a.validationErrors,
		a.requestsInFlight,
		a.requestSize,
		a.responseSize,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// RecordRequest records an API request
func (a *APIMetrics) RecordRequest(resource, verb, statusCode string, duration time.Duration) {
	a.requestsTotal.WithLabelValues(resource, verb, statusCode).Inc()
	a.requestDuration.WithLabelValues(resource, verb).Observe(duration.Seconds())
}

// SetActiveWatchers sets the number of active watchers for a resource
func (a *APIMetrics) SetActiveWatchers(resource string, count int) {
	a.watchersActive.WithLabelValues(resource).Set(float64(count))
}

// RecordValidationError records a validation error
func (a *APIMetrics) RecordValidationError(resource, operation string) {
	a.validationErrors.WithLabelValues(resource, operation).Inc()
}

// IncrementInflightRequests increments the in-flight request gauge
func (a *APIMetrics) IncrementInflightRequests(resource, verb string) {
	a.requestsInFlight.WithLabelValues(resource, verb).Inc()
}

// DecrementInflightRequests decrements the in-flight request gauge
func (a *APIMetrics) DecrementInflightRequests(resource, verb string) {
	a.requestsInFlight.WithLabelValues(resource, verb).Dec()
}

// ObserveRequestSize records the size of an API request payload
func (a *APIMetrics) ObserveRequestSize(resource, verb string, sizeBytes int) {
	a.requestSize.WithLabelValues(resource, verb).Observe(float64(sizeBytes))
}

// ObserveResponseSize records the size of an API response payload
func (a *APIMetrics) ObserveResponseSize(resource, verb string, sizeBytes int) {
	a.responseSize.WithLabelValues(resource, verb).Observe(float64(sizeBytes))
}
