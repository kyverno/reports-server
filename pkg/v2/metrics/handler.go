package metrics

import (
	"net/http"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// CreateHandler creates an HTTP handler for Prometheus metrics
func CreateHandler() (http.HandlerFunc, error) {
	// Create Kubernetes metrics registry
	kubeRegistry := metrics.NewKubeRegistry()

	// Create and register our metrics
	registry := NewRegistry()
	if err := registry.Register(kubeRegistry); err != nil {
		return nil, err
	}

	// Return handler that serves both legacy and custom metrics
	return func(w http.ResponseWriter, req *http.Request) {
		// Serve legacy metrics (apiserver, go runtime, process stats, etc.)
		legacyregistry.Handler().ServeHTTP(w, req)

		// Serve custom metrics (reports-server specific)
		metrics.HandlerFor(kubeRegistry, metrics.HandlerOpts{}).ServeHTTP(w, req)
	}, nil
}

// MustCreateHandler creates a metrics handler and panics on error
// Use this in init or main where failure should be fatal
func MustCreateHandler() http.HandlerFunc {
	handler, err := CreateHandler()
	if err != nil {
		panic("failed to create metrics handler: " + err.Error())
	}
	return handler
}
