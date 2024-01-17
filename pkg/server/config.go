package server

import (
	"net/http"

	"github.com/kyverno/policy-reports/pkg/api"
	"github.com/kyverno/policy-reports/pkg/storage"
	"github.com/kyverno/policy-reports/pkg/storage/db"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // for client-go metrics registration
)

type Config struct {
	Apiserver *genericapiserver.Config
	Rest      *rest.Config
	Debug     bool
	DBconfig  *db.PostgresConfig
}

func (c Config) Complete() (*server, error) {
	// Disable default metrics handler and create custom one
	c.Apiserver.EnableMetrics = false
	metricsHandler, err := c.metricsHandler()
	if err != nil {
		return nil, err
	}
	genericServer, err := c.Apiserver.Complete(nil).New("policy-reports", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}
	genericServer.Handler.NonGoRestfulMux.HandleFunc("/metrics", metricsHandler)

	store, err := storage.New(c.Debug, c.DBconfig)
	if err != nil {
		return nil, err
	}
	if err := api.Install(store, genericServer); err != nil {
		return nil, err
	}

	s := NewServer(
		genericServer,
		store,
	)
	err = s.RegisterProbes()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c Config) metricsHandler() (http.HandlerFunc, error) {
	// Create registry for Policy Server metrics
	registry := metrics.NewKubeRegistry()
	err := RegisterMetrics(registry)
	if err != nil {
		return nil, err
	}
	// Register apiserver metrics in legacy registry
	apimetrics.Register()

	// Return handler that serves metrics from both legacy and Metrics Server registry
	return func(w http.ResponseWriter, req *http.Request) {
		legacyregistry.Handler().ServeHTTP(w, req)
		metrics.HandlerFor(registry, metrics.HandlerOpts{}).ServeHTTP(w, req)
	}, nil
}
