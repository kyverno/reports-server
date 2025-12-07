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
}

// NewAPIMetrics creates API metrics collectors
func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{
		requestsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: "reports_server",
				Subsystem: "api",
				Name:      "requests_total",
				Help:      "Total number of API requests",
			},
			[]string{"resource", "verb", "status_code"},
		),

		requestDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: "reports_server",
				Subsystem: "api",
				Name:      "request_duration_seconds",
				Help:      "Duration of API requests",
				Buckets:   metrics.DefBuckets,
			},
			[]string{"resource", "verb"},
		),

		watchersActive: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: "reports_server",
				Subsystem: "api",
				Name:      "watchers_active",
				Help:      "Number of active watchers",
			},
			[]string{"resource"},
		),

		validationErrors: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: "reports_server",
				Subsystem: "api",
				Name:      "validation_errors_total",
				Help:      "Total number of validation errors",
			},
			[]string{"resource", "operation"},
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
