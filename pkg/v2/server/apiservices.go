package server

import (
	"context"
	"fmt"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1client "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

// APIServiceConfig holds configuration for a single APIService
type APIServiceConfig struct {
	Name             string
	Group            string
	Version          string
	ServiceName      string
	ServiceNamespace string
	Enabled          bool
}

// InstallAPIServices creates or updates APIService objects in Kubernetes
// This tells Kubernetes to route API requests to our aggregated API server
func (c *Config) InstallAPIServices() error {
	if c.RESTConfig == nil {
		klog.V(logging.LevelWarning).Info("No REST config provided, skipping APIService installation")
		return nil
	}

	klog.V(logging.LevelInfo).Info("Installing APIServices")
	managedCount := 0

	// Create API registration client
	apiRegClient, err := apiregistrationv1client.NewForConfig(c.RESTConfig)
	if err != nil {
		return fmt.Errorf("failed to create API registration client: %w", err)
	}

	// Define APIService configurations
	apiServices := []APIServiceConfig{
		{
			Name:             "v1alpha2.wgpolicyk8s.io",
			Group:            GroupWGPolicy,
			Version:          "v1alpha2",
			ServiceName:      c.Server.ServiceName,
			ServiceNamespace: c.Server.Namespace,
			Enabled:          c.Server.EnablePolicyReports,
		},
		{
			Name:             "v1.reports.kyverno.io",
			Group:            GroupKyvernoReports,
			Version:          "v1",
			ServiceName:      c.Server.ServiceName,
			ServiceNamespace: c.Server.Namespace,
			Enabled:          c.Server.EnableEphemeralReports,
		},
		{
			Name:             "v1alpha1.openreports.io",
			Group:            GroupOpenReports,
			Version:          "v1alpha1",
			ServiceName:      c.Server.ServiceName,
			ServiceNamespace: c.Server.Namespace,
			Enabled:          c.Server.EnableOpenReports,
		},
	}

	// Create or update each APIService
	for _, svcCfg := range apiServices {
		if err := c.ensureAPIService(apiRegClient, svcCfg); err != nil {
			return fmt.Errorf("failed to ensure APIService %s: %w", svcCfg.Name, err)
		}
		if svcCfg.Enabled {
			managedCount++
		}
	}

	// Record metrics
	metrics.Global().Server().SetAPIServicesManaged(managedCount)

	klog.V(logging.LevelInfo).InfoS("APIServices installation complete",
		"managed", managedCount)

	return nil
}

// ensureAPIService creates or updates an APIService, or deletes it if disabled
func (c *Config) ensureAPIService(
	client *apiregistrationv1client.ApiregistrationV1Client,
	cfg APIServiceConfig,
) error {
	ctx := context.TODO()

	// Check if APIService already exists
	existing, err := client.APIServices().Get(ctx, cfg.Name, metav1.GetOptions{})

	if cfg.Enabled {
		// Build desired APIService
		desired := buildAPIService(cfg)

		if errors.IsNotFound(err) {
			// Create new APIService
			_, err := client.APIServices().Create(ctx, desired, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create APIService: %w", err)
			}
			klog.V(logging.LevelInfo).InfoS("Created APIService", "name", cfg.Name, "group", cfg.Group)
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to get APIService: %w", err)
		}

		// Update if service reference changed
		if needsUpdate(existing, desired) {
			desired.SetResourceVersion(existing.GetResourceVersion())
			_, err := client.APIServices().Update(ctx, desired, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update APIService: %w", err)
			}
			klog.V(logging.LevelInfo).InfoS("Updated APIService", "name", cfg.Name, "group", cfg.Group)
		} else {
			klog.V(logging.LevelDebug).InfoS("APIService already up to date", "name", cfg.Name)
		}

		return nil
	}

	// Disabled - delete if exists
	if err == nil {
		klog.V(logging.LevelInfo).InfoS("APIService disabled, deleting", "name", cfg.Name)
		if err := client.APIServices().Delete(ctx, cfg.Name, metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete APIService: %w", err)
			}
		}
		klog.V(logging.LevelInfo).InfoS("Deleted APIService", "name", cfg.Name)
	}

	return nil
}

// buildAPIService constructs an APIService object from configuration
func buildAPIService(cfg APIServiceConfig) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "reports-server",
				"app.kubernetes.io/component":  "apiservice",
			},
		},
		Spec: apiregistrationv1.APIServiceSpec{
			Group:                 cfg.Group,
			Version:               cfg.Version,
			GroupPriorityMinimum:  100,
			VersionPriority:       100,
			InsecureSkipTLSVerify: true,
			Service: &apiregistrationv1.ServiceReference{
				Name:      cfg.ServiceName,
				Namespace: cfg.ServiceNamespace,
			},
		},
	}
}

// needsUpdate checks if an APIService needs to be updated
func needsUpdate(existing, desired *apiregistrationv1.APIService) bool {
	// Check if service reference changed
	if existing.Spec.Service == nil || desired.Spec.Service == nil {
		return true
	}

	return existing.Spec.Service.Name != desired.Spec.Service.Name ||
		existing.Spec.Service.Namespace != desired.Spec.Service.Namespace
}

// CleanupAPIServices removes all APIServices created by this server
// This should be called during graceful shutdown
func (c *Config) CleanupAPIServices() error {
	if c.RESTConfig == nil {
		return nil
	}

	klog.Info("Cleaning up APIServices")

	apiRegClient, err := apiregistrationv1client.NewForConfig(c.RESTConfig)
	if err != nil {
		return fmt.Errorf("failed to create API registration client: %w", err)
	}

	ctx := context.TODO()

	apiServiceNames := []string{
		"v1alpha2.wgpolicyk8s.io",
		"v1.reports.kyverno.io",
		"v1alpha1.openreports.io",
	}

	var errs []error
	for _, name := range apiServiceNames {
		if err := apiRegClient.APIServices().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete APIService %s: %w", name, err))
			}
		} else {
			klog.InfoS("Deleted APIService", "name", name)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}

	return nil
}
