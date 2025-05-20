package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
)

var (
	dbRequestTotalMetrics = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Namespace: "reports_server",
			Subsystem: "storage",
			Name:      "db_requests_total",
			Help:      "Total number of db requests",
		},
		[]string{"type", "operation", "reportType"},
	)
	dbRequestLatencyMetrics = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Namespace: "reports_server",
			Subsystem: "storage",
			Name:      "db_request_duration_seconds",
			Help:      "duration of db requests",
			Buckets:   metrics.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"type", "operation", "reportType"},
	)
)

func RegisterServerMetrics(registrationFunc func(metrics.Registerable) error) error {
	err := registrationFunc(dbRequestTotalMetrics)
	if err != nil {
		return err
	}
	err = registrationFunc(dbRequestLatencyMetrics)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDBRequestTotalMetrics(dbType string, operation string, reportType string) {
	dbRequestTotalMetrics.WithLabelValues(dbType, operation, reportType).Inc()
}

func UpdateDBRequestLatencyMetrics(dbType string, operation string, reportType string, duration time.Duration) {
	dbRequestLatencyMetrics.WithLabelValues(dbType, operation, reportType).Observe(duration.Seconds())
}
