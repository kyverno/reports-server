package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kyverno/reports-server/pkg/storage"
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
	storage storage.Interface,
) *server {
	return &server{
		GenericAPIServer: apiserver,
		storage:          storage,
	}
}

type server struct {
	*genericapiserver.GenericAPIServer
	storage storage.Interface
}

// RunUntil starts the background scraping goroutine and runs the apiserver serving metrics.
func (s *server) RunUntil(stopCh <-chan struct{}) error {
	// Create a context that will be canceled when stopCh is closed.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopCh
		cancel()
	}()
	return s.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

func (s *server) RegisterProbes() error {
	err := s.AddReadyzChecks(s.probeMetricStorageReady("policy-db-ready"))
	if err != nil {
		return err
	}
	return nil
}

func (s *server) probeMetricStorageReady(name string) healthz.HealthChecker {
	return healthz.NamedCheck(name, func(r *http.Request) error {
		if !s.storage.Ready() {
			err := fmt.Errorf("db not working")
			klog.InfoS("Failed probe", "probe", name, "err", err)
			return err
		}
		return nil
	})
}
