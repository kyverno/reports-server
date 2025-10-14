package metrics

import (
	"k8s.io/component-base/metrics"
)

// WatchMetrics tracks watch-related metrics
type WatchMetrics struct {
	connectionsActive *metrics.GaugeVec
	connectionsTotal  *metrics.CounterVec
	eventsTotal       *metrics.CounterVec
	eventsSent        *metrics.CounterVec
	eventsDropped     *metrics.CounterVec
	connectionDuration *metrics.HistogramVec
}

// NewWatchMetrics creates watch metrics collectors
func NewWatchMetrics() *WatchMetrics {
	return &WatchMetrics{
		connectionsActive: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "connections_active",
				Help:      "Number of active watch connections",
			},
			[]string{"resource"},
		),

		connectionsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "connections_total",
				Help:      "Total number of watch connections",
			},
			[]string{"resource"},
		),

		eventsTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "events_total",
				Help:      "Total number of watch events generated",
			},
			[]string{"resource", "event_type"},
		),

		eventsSent: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "events_sent",
				Help:      "Total number of watch events sent to clients",
			},
			[]string{"resource"},
		),

		eventsDropped: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "events_dropped",
				Help:      "Total number of watch events dropped",
			},
			[]string{"resource", "reason"},
		),

		connectionDuration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Namespace: Namespace,
				Subsystem: SubsystemWatch,
				Name:      "connection_duration_seconds",
				Help:      "Duration of watch connections",
				Buckets:   metrics.ExponentialBuckets(1, 2, 15), // 1s to ~9 hours
			},
			[]string{"resource"},
		),
	}
}

// Register registers all watch metrics
func (w *WatchMetrics) Register(registry metrics.KubeRegistry) error {
	collectors := []metrics.Registerable{
		w.connectionsActive,
		w.connectionsTotal,
		w.eventsTotal,
		w.eventsSent,
		w.eventsDropped,
		w.connectionDuration,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// IncrementActiveConnections increments active watch connections
func (w *WatchMetrics) IncrementActiveConnections(resource string) {
	w.connectionsActive.WithLabelValues(resource).Inc()
}

// DecrementActiveConnections decrements active watch connections
func (w *WatchMetrics) DecrementActiveConnections(resource string) {
	w.connectionsActive.WithLabelValues(resource).Dec()
}

// RecordConnection records a new watch connection
func (w *WatchMetrics) RecordConnection(resource string) {
	w.connectionsTotal.WithLabelValues(resource).Inc()
}

// RecordEvent records a watch event generation
func (w *WatchMetrics) RecordEvent(resource, eventType string) {
	w.eventsTotal.WithLabelValues(resource, eventType).Inc()
}

// RecordEventSent records a watch event sent to a client
func (w *WatchMetrics) RecordEventSent(resource string) {
	w.eventsSent.WithLabelValues(resource).Inc()
}

// RecordEventDropped records a dropped watch event
func (w *WatchMetrics) RecordEventDropped(resource, reason string) {
	w.eventsDropped.WithLabelValues(resource, reason).Inc()
}

// ObserveConnectionDuration records the duration of a watch connection
func (w *WatchMetrics) ObserveConnectionDuration(resource string, durationSeconds float64) {
	w.connectionDuration.WithLabelValues(resource).Observe(durationSeconds)
}

