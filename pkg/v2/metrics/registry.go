package metrics

import (
	"fmt"

	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/component-base/metrics"
)

// Registry holds all metrics collectors
type Registry struct {
	storage *StorageMetrics
	api     *APIMetrics
}

// NewRegistry creates a new metrics registry with all collectors
func NewRegistry() *Registry {
	return &Registry{
		storage: NewStorageMetrics(),
		api:     NewAPIMetrics(),
	}
}

// Storage returns the storage metrics collector
func (r *Registry) Storage() *StorageMetrics {
	return r.storage
}

// API returns the API metrics collector
func (r *Registry) API() *APIMetrics {
	return r.api
}

// Register registers all metrics with the Kubernetes registry
func (r *Registry) Register(kubeRegistry metrics.KubeRegistry) error {
	// Register storage metrics
	if err := r.storage.Register(kubeRegistry); err != nil {
		return fmt.Errorf("failed to register storage metrics: %w", err)
	}

	// Register API metrics
	if err := r.api.Register(kubeRegistry); err != nil {
		return fmt.Errorf("failed to register API metrics: %w", err)
	}

	// Register API server metrics in legacy registry
	apimetrics.Register()

	return nil
}
