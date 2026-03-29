package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kyverno/reports-server/pkg/storage/api"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/component-base/metrics"
	"k8s.io/klog/v2"
)

func RegisterServerMetrics(registrationFunc func(metrics.Registerable) error) error {
	metricFreshness := metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Namespace: "reports_server",
			Subsystem: "storage",
			Name:      "policy_reports_fetch_time",
			Help:      "Time taken to fetch reports",
			Buckets:   metrics.ExponentialBuckets(1, 1.364, 20),
		},
		[]string{},
	)
	return registrationFunc(metricFreshness)
}

func NewServer(
	apiserver *genericapiserver.GenericAPIServer,
	storage api.Storage,
) *server {
	return &server{
		GenericAPIServer: apiserver,
		storage:          storage,
	}
}

type server struct {
	*genericapiserver.GenericAPIServer
	storage api.Storage
}

// RunUntil starts background scraping goroutine and runs apiserver serving metrics.
func (s *server) RunUntil(stopCh <-chan struct{}) error {
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}

func (s *server) RegisterProbes(readinessTimeout time.Duration) error {
	err := s.AddReadyzChecks(s.probeMetricStorageReady("policy-db-ready", readinessTimeout))
	if err != nil {
		return err
	}
	return nil
}

func (s *server) probeMetricStorageReady(name string, readinessTimeout time.Duration) healthz.HealthChecker {
	return healthz.NamedCheck(name, func(r *http.Request) error {
		ctx, cancel := context.WithTimeout(context.Background(), readinessTimeout)
		defer cancel()
		if !s.storage.Ready(ctx) {
			err := fmt.Errorf("db not working")
			klog.InfoS("Failed probe", "probe", name, "err", err)
			return err
		}
		return nil
	})
}
