package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/klog/v2"
)

// StorageHealthChecker checks if storage is accessible
type StorageHealthChecker struct {
	name         string
	repositories *Repositories
}

// NewStorageHealthChecker creates a new storage health checker
func NewStorageHealthChecker(name string, repos *Repositories) healthz.HealthChecker {
	return &StorageHealthChecker{
		name:         name,
		repositories: repos,
	}
}

// Name returns the name of the health check
func (s *StorageHealthChecker) Name() string {
	return s.name
}

// Check verifies storage is accessible by attempting a list operation
func (s *StorageHealthChecker) Check(req *http.Request) error {
	ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()

	// Try to list from one of the repositories to verify storage works
	if s.repositories.PolicyReports != nil {
		_, err := s.repositories.PolicyReports.List(ctx, storage.Filter{})
		if err != nil {
			klog.V(4).InfoS("Storage health check failed", "error", err)
			return fmt.Errorf("storage not accessible: %w", err)
		}
	}

	return nil
}

// PingHealthChecker is a simple health check that always succeeds
type PingHealthChecker struct {
	name string
}

// NewPingHealthChecker creates a health checker that always succeeds
func NewPingHealthChecker(name string) healthz.HealthChecker {
	return &PingHealthChecker{name: name}
}

// Name returns the name of the health check
func (p *PingHealthChecker) Name() string {
	return p.name
}

// Check always returns nil (success)
func (p *PingHealthChecker) Check(req *http.Request) error {
	return nil
}
