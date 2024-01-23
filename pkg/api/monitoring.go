package api

import (
	"k8s.io/component-base/metrics"
)

var metricFreshness = metrics.NewHistogramVec(
	&metrics.HistogramOpts{
		Namespace: "reports_server",
		Subsystem: "api",
		Name:      "reports_server_export_time",
		Help:      "serve of reports exported",
		Buckets:   metrics.ExponentialBuckets(1, 1.364, 20),
	},
	[]string{},
)

func RegisterAPIMetrics(registrationFunc func(metrics.Registerable) error) error {
	return registrationFunc(metricFreshness)
}
