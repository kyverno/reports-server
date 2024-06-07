package server

import (
	"context"
	"net/http"

	"github.com/nirmata/reports-server/pkg/api"
	"github.com/nirmata/reports-server/pkg/storage"
	"github.com/nirmata/reports-server/pkg/storage/db"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
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
	genericServer, err := c.Apiserver.Complete(nil).New("reports-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}
	genericServer.Handler.NonGoRestfulMux.HandleFunc("/metrics", metricsHandler)

	id, err := c.getClusterId()
	if err != nil {
		return nil, err
	}
	store, err := storage.New(c.Debug, c.DBconfig, id)
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

func (c Config) getClusterId() (string, error) {
	clientset, err := kubernetes.NewForConfig(c.Rest)
	if err != nil {
		return "", err
	}

	// Kubernetes clusters do not have a uid. The uid of kubesystem namespace does not change and is commonly accepted as the id of the cluster
	ns, err := clientset.CoreV1().Namespaces().Get(context.Background(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(ns.GetUID()), nil
}
