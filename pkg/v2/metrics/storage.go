package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
)

// StorageMetrics tracks storage-related metrics
type StorageMetrics struct {
	fetchDuration      *metrics.HistogramVec
	reportsTotal       *metrics.GaugeVec
	operationsTotal    *metrics.CounterVec
	operationsDuration *metrics.HistogramVec
}

// NewStorageMetrics creates storage metrics collectors
func NewStorageMetrics() *StorageMetrics {
	return &StorageMetrics{
		fetchDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: "reports_server",
				Subsystem: "storage",
				Name:      "fetch_duration_seconds",
				Help:      "Time taken to fetch reports from storage",
				Buckets:   metrics.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
			},
			[]string{"resource_type", "operation"},
		),

		reportsTotal: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: "reports_server",
				Subsystem: "storage",
				Name:      "reports_total",
				Help:      "Total number of reports in storage",
			},
			[]string{"resource_type", "cluster_id"},
		),

		operationsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: "reports_server",
				Subsystem: "storage",
				Name:      "operations_total",
				Help:      "Total number of storage operations",
			},
			[]string{"resource_type", "operation", "status"},
		),

		operationsDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: "reports_server",
				Subsystem: "storage",
				Name:      "operation_duration_seconds",
				Help:      "Duration of storage operations",
				Buckets:   metrics.ExponentialBuckets(0.001, 2, 15),
			},
			[]string{"resource_type", "operation"},
		),
	}
}

// Register registers all storage metrics
func (s *StorageMetrics) Register(registry metrics.KubeRegistry) error {
	collectors := []metrics.Registerable{
		s.fetchDuration,
		s.reportsTotal,
		s.operationsTotal,
		s.operationsDuration,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// ObserveFetch records the time taken to fetch reports
func (s *StorageMetrics) ObserveFetch(resourceType, operation string, duration time.Duration) {
	s.fetchDuration.WithLabelValues(resourceType, operation).Observe(duration.Seconds())
}

// SetReportsTotal sets the total number of reports for a resource type
func (s *StorageMetrics) SetReportsTotal(resourceType, clusterID string, count int) {
	s.reportsTotal.WithLabelValues(resourceType, clusterID).Set(float64(count))
}

// RecordOperation records a storage operation
func (s *StorageMetrics) RecordOperation(resourceType, operation, status string, duration time.Duration) {
	s.operationsTotal.WithLabelValues(resourceType, operation, status).Inc()
	s.operationsDuration.WithLabelValues(resourceType, operation).Observe(duration.Seconds())
}
