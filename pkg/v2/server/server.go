package server

import (
	"context"
	"fmt"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/klog/v2"
)

// Server is the v2 reports server
type Server struct {
	// GenericAPIServer is the underlying Kubernetes API server
	GenericAPIServer *genericapiserver.GenericAPIServer

	// repositories holds all storage repositories
	repositories *Repositories

	// config is the server configuration
	config *Config
}

// Run starts the server and blocks until the stop channel is closed
func (s *Server) Run(ctx context.Context) error {
	serverStartTime := time.Now()
	
	klog.V(logging.LevelInfo).InfoS("Starting reports-server v2",
		"storageBackend", s.config.Storage.Backend,
		"enabledAPIGroups", s.countEnabledAPIGroups(),
	)

	// Install API groups
	if err := s.InstallAPIGroups(); err != nil {
		return fmt.Errorf("failed to install API groups: %w", err)
	}

	// Install health checks
	if err := s.InstallHealthChecks(); err != nil {
		return fmt.Errorf("failed to install health checks: %w", err)
	}

	// Track uptime
	go s.trackUptime(ctx, serverStartTime)

	// Setup graceful shutdown
	go func() {
		<-ctx.Done()
		klog.V(logging.LevelInfo).Info("Shutdown signal received, cleaning up...")
		s.Shutdown()
	}()

	// Start the generic API server
	preparedServer := s.GenericAPIServer.PrepareRun()
	
	klog.V(logging.LevelInfo).InfoS("Reports-server v2 is ready",
		"address", s.GenericAPIServer.LoopbackClientConfig.Host,
		"uptime", time.Since(serverStartTime),
	)
	
	return preparedServer.Run(ctx.Done())
}

// trackUptime periodically updates the uptime metric
func (s *Server) trackUptime(ctx context.Context, startTime time.Time) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.Global().Server().SetUptime(time.Since(startTime).Seconds())
		}
	}
}

// countEnabledAPIGroups counts how many API groups are enabled
func (s *Server) countEnabledAPIGroups() int {
	count := 0
	if s.config.Server.EnablePolicyReports {
		count++
	}
	if s.config.Server.EnableEphemeralReports {
		count++
	}
	if s.config.Server.EnableOpenReports {
		count++
	}
	return count
}

// Shutdown performs graceful shutdown of the server
func (s *Server) Shutdown() {
	klog.Info("Shutting down reports-server v2")

	// Cleanup APIServices (make them local again)
	if err := s.config.CleanupAPIServices(); err != nil {
		klog.ErrorS(err, "Failed to cleanup APIServices during shutdown")
	}

	klog.Info("Shutdown complete")
}

// InstallAPIGroups installs all configured API groups
func (s *Server) InstallAPIGroups() error {
	klog.V(logging.LevelInfo).Info("Installing API groups")
	installedCount := 0

	// Install wgpolicyk8s.io (PolicyReports, ClusterPolicyReports)
	if s.config.Server.EnablePolicyReports {
		if err := s.installPolicyReportsAPI(); err != nil {
			return fmt.Errorf("failed to install policy reports API: %w", err)
		}
		installedCount++
		klog.V(logging.LevelInfo).InfoS("Installed API group",
			"group", "wgpolicyk8s.io",
			"version", "v1alpha2",
			"resources", 2,
		)
	}

	// Install reports.kyverno.io (EphemeralReports, ClusterEphemeralReports)
	if s.config.Server.EnableEphemeralReports {
		if err := s.installEphemeralReportsAPI(); err != nil {
			return fmt.Errorf("failed to install ephemeral reports API: %w", err)
		}
		installedCount++
		klog.V(logging.LevelInfo).InfoS("Installed API group",
			"group", "reports.kyverno.io",
			"version", "v1",
			"resources", 2,
		)
	}

	// Install openreports.io (Reports, ClusterReports)
	if s.config.Server.EnableOpenReports {
		if err := s.installOpenReportsAPI(); err != nil {
			return fmt.Errorf("failed to install open reports API: %w", err)
		}
		installedCount++
		klog.V(logging.LevelInfo).InfoS("Installed API group",
			"group", "openreports.io",
			"version", "v1alpha1",
			"resources", 2,
		)
	}

	// Record metric
	metrics.Global().Server().SetAPIGroupsInstalled(installedCount)

	klog.V(logging.LevelInfo).InfoS("API groups installation complete",
		"totalInstalled", installedCount)

	return nil
}

// installPolicyReportsAPI installs the wgpolicyk8s.io API group
func (s *Server) installPolicyReportsAPI() error {
	// Create handler factory
	factory := NewHandlerFactory(s.config.Versioning)

	// Create REST handlers for PolicyReports and ClusterPolicyReports
	polrHandler := factory.CreatePolicyReportHandler(s.repositories.PolicyReports)
	cpolrHandler := factory.CreateClusterPolicyReportHandler(s.repositories.ClusterPolicyReports)

	// Build resources map
	resources := map[string]rest.Storage{
		"policyreports":        polrHandler,
		"clusterpolicyreports": cpolrHandler,
	}

	// Create API group info
	apiGroupInfo := BuildAPIGroupInfo(
		"wgpolicyk8s.io",
		"v1alpha2",
		resources,
		GetScheme(),
		GetCodecs(),
	)

	// Install the API group
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}

// installEphemeralReportsAPI installs the reports.kyverno.io API group
func (s *Server) installEphemeralReportsAPI() error {
	// Create handler factory
	factory := NewHandlerFactory(s.config.Versioning)

	// Create REST handlers
	ephrHandler := factory.CreateEphemeralReportHandler(s.repositories.EphemeralReports)
	cephrHandler := factory.CreateClusterEphemeralReportHandler(s.repositories.ClusterEphemeralReports)

	// Build resources map
	resources := map[string]rest.Storage{
		"ephemeralreports":        ephrHandler,
		"clusterephemeralreports": cephrHandler,
	}

	// Create API group info
	apiGroupInfo := BuildAPIGroupInfo(
		"reports.kyverno.io",
		"v1",
		resources,
		GetScheme(),
		GetCodecs(),
	)

	// Install the API group
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}

// installOpenReportsAPI installs the openreports.io API group
func (s *Server) installOpenReportsAPI() error {
	// Create handler factory
	factory := NewHandlerFactory(s.config.Versioning)

	// Create REST handlers
	repHandler := factory.CreateReportHandler(s.repositories.Reports)
	crepHandler := factory.CreateClusterReportHandler(s.repositories.ClusterReports)

	// Build resources map
	resources := map[string]rest.Storage{
		"reports":        repHandler,
		"clusterreports": crepHandler,
	}

	// Create API group info
	apiGroupInfo := BuildAPIGroupInfo(
		"openreports.io",
		"v1alpha1",
		resources,
		GetScheme(),
		GetCodecs(),
	)

	// Install the API group
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}

// InstallHealthChecks installs health check endpoints
func (s *Server) InstallHealthChecks() error {
	klog.V(logging.LevelInfo).Info("Installing health checks")

	// Readiness check - verifies storage is accessible
	storageCheck := NewStorageHealthChecker("storage-ready", s.repositories)
	if err := s.GenericAPIServer.AddReadyzChecks(storageCheck); err != nil {
		return fmt.Errorf("failed to add readiness check: %w", err)
	}

	// Liveness check - simple ping to verify server is alive
	pingCheck := NewPingHealthChecker("ping")
	if err := s.GenericAPIServer.AddHealthChecks(pingCheck); err != nil {
		return fmt.Errorf("failed to add liveness check: %w", err)
	}

	// Initialize health check metrics
	metrics.Global().Server().SetHealthCheckStatus("storage-ready", true)
	metrics.Global().Server().SetHealthCheckStatus("ping", true)

	klog.V(logging.LevelInfo).InfoS("Health checks installed",
		"checks", 2,
	)
	return nil
}
