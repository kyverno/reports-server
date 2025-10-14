package metrics

import (
	"k8s.io/component-base/metrics"
)

// ServerMetrics tracks server-level metrics
type ServerMetrics struct {
	uptime              *metrics.GaugeVec
	apiGroupsInstalled  *metrics.GaugeVec
	healthCheckStatus   *metrics.GaugeVec
	apiServicesManaged  *metrics.GaugeVec
	configReloadTotal   *metrics.CounterVec
}

// NewServerMetrics creates server metrics collectors
func NewServerMetrics() *ServerMetrics {
	return &ServerMetrics{
		uptime: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemServer,
				Name:      "uptime_seconds",
				Help:      "Server uptime in seconds",
			},
			[]string{},
		),

		apiGroupsInstalled: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemServer,
				Name:      "api_groups_installed",
				Help:      "Number of API groups installed",
			},
			[]string{},
		),

		healthCheckStatus: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemServer,
				Name:      "health_check_status",
				Help:      "Health check status (1=healthy, 0=unhealthy)",
			},
			[]string{"check_name"},
		),

		apiServicesManaged: metrics.NewGaugeVec(
			&metrics.GaugeOpts{
				Namespace: Namespace,
				Subsystem: SubsystemServer,
				Name:      "apiservices_managed",
				Help:      "Number of APIServices managed by this server",
			},
			[]string{},
		),

		configReloadTotal: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Namespace: Namespace,
				Subsystem: SubsystemServer,
				Name:      "config_reload_total",
				Help:      "Total number of configuration reloads",
			},
			[]string{"status"},
		),
	}
}

// Register registers all server metrics
func (s *ServerMetrics) Register(registry metrics.KubeRegistry) error {
	collectors := []metrics.Registerable{
		s.uptime,
		s.apiGroupsInstalled,
		s.healthCheckStatus,
		s.apiServicesManaged,
		s.configReloadTotal,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// SetUptime sets the server uptime in seconds
func (s *ServerMetrics) SetUptime(seconds float64) {
	s.uptime.WithLabelValues().Set(seconds)
}

// SetAPIGroupsInstalled sets the number of API groups installed
func (s *ServerMetrics) SetAPIGroupsInstalled(count int) {
	s.apiGroupsInstalled.WithLabelValues().Set(float64(count))
}

// SetHealthCheckStatus sets health check status (1=healthy, 0=unhealthy)
func (s *ServerMetrics) SetHealthCheckStatus(checkName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	s.healthCheckStatus.WithLabelValues(checkName).Set(value)
}

// SetAPIServicesManaged sets the number of APIServices managed
func (s *ServerMetrics) SetAPIServicesManaged(count int) {
	s.apiServicesManaged.WithLabelValues().Set(float64(count))
}

// RecordConfigReload records a configuration reload attempt
func (s *ServerMetrics) RecordConfigReload(status string) {
	s.configReloadTotal.WithLabelValues(status).Inc()
}

