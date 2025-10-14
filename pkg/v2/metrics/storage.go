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
	connectionPool     *metrics.GaugeVec
	filterEfficiency   *metrics.HistogramVec
	cacheHitRate       *metrics.CounterVec
}

// NewStorageMetrics creates storage metrics collectors
func NewStorageMetrics() *StorageMetrics {
	return &StorageMetrics{
		fetchDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "fetch_duration_seconds",
				Help:      "Time taken to fetch reports from storage",
				Buckets:   metrics.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
			},
			[]string{"resource_type", "operation"},
		),

		reportsTotal: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "reports_total",
				Help:      "Total number of reports in storage",
			},
			[]string{"resource_type", "cluster_id"},
		),

		operationsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "operations_total",
				Help:      "Total number of storage operations",
			},
			[]string{"resource_type", "operation", "status"},
		),

		operationsDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "operation_duration_seconds",
				Help:      "Duration of storage operations",
				Buckets:   metrics.ExponentialBuckets(0.001, 2, 15),
			},
			[]string{"resource_type", "operation"},
		),

		connectionPool: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "connection_pool_size",
				Help:      "Current database connection pool size",
			},
			[]string{"backend", "pool_type"}, // backend: postgres/etcd, pool_type: active/idle
		),

		filterEfficiency: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "filter_efficiency_ratio",
				Help:      "Ratio of matched items to total items (0.0-1.0)",
				Buckets:   metrics.LinearBuckets(0, 0.1, 11), // 0.0, 0.1, ..., 1.0
			},
			[]string{"resource_type"},
		),

		cacheHitRate: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemStorage,
				Name:      "cache_requests_total",
				Help:      "Total number of cache requests",
			},
			[]string{"resource_type", "result"}, // result: hit/miss
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
		s.connectionPool,
		s.filterEfficiency,
		s.cacheHitRate,
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

// SetConnectionPoolSize sets the database connection pool size
func (s *StorageMetrics) SetConnectionPoolSize(backend, poolType string, size int) {
	s.connectionPool.WithLabelValues(backend, poolType).Set(float64(size))
}

// ObserveFilterEfficiency records how efficient filtering was
// ratio = matched items / total items (0.0 to 1.0)
func (s *StorageMetrics) ObserveFilterEfficiency(resourceType string, matched, total int) {
	if total > 0 {
		ratio := float64(matched) / float64(total)
		s.filterEfficiency.WithLabelValues(resourceType).Observe(ratio)
	}
}

// RecordCacheRequest records a cache hit or miss
func (s *StorageMetrics) RecordCacheRequest(resourceType string, hit bool) {
	result := "miss"
	if hit {
		result = "hit"
	}
	s.cacheHitRate.WithLabelValues(resourceType, result).Inc()
}
