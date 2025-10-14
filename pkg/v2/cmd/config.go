package cmd

import (
	"fmt"
	"net"

	generatedopenapi "github.com/kyverno/reports-server/pkg/api/generated/openapi"
	"github.com/kyverno/reports-server/pkg/v2/server"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/pkg/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Complete fills in missing values with defaults and creates server config
func (o *Options) Complete() (*server.Config, error) {
	// Create server components
	genericConfig, err := o.buildGenericConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create generic config: %w", err)
	}

	restConfig, err := o.buildRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config: %w", err)
	}

	storageConfig, err := o.buildStorageConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage config: %w", err)
	}

	serverConfig := o.buildServerConfig()

	return &server.Config{
		GenericConfig: genericConfig,
		RESTConfig:    restConfig,
		Storage:       storageConfig,
		Server:        serverConfig,
		Versioning:    nil, // Created by server during Complete()
	}, nil
}

// buildGenericConfig creates the Kubernetes API server configuration
func (o *Options) buildGenericConfig() (*genericapiserver.RecommendedConfig, error) {
	// Setup TLS certificates
	if err := o.setupTLS(); err != nil {
		return nil, err
	}

	// Create base configuration
	config := genericapiserver.NewRecommendedConfig(server.Codecs)

	// Apply server options
	if err := o.applyServerOptions(config); err != nil {
		return nil, err
	}

	// Apply authentication and authorization
	if err := o.applyAuthOptions(config); err != nil {
		return nil, err
	}

	// Configure OpenAPI
	o.configureOpenAPI(config)

	return config, nil
}

// setupTLS creates self-signed certificates if none provided
func (o *Options) setupTLS() error {
	return o.SecureServing.MaybeDefaultWithSelfSignedCerts(
		"localhost",
		nil,
		[]net.IP{net.ParseIP("127.0.0.1")},
	)
}

// applyServerOptions applies secure serving and audit options
func (o *Options) applyServerOptions(config *genericapiserver.RecommendedConfig) error {
	// Apply secure serving
	if err := o.SecureServing.ApplyTo(&config.SecureServing, &config.LoopbackClientConfig); err != nil {
		return fmt.Errorf("failed to apply secure serving: %w", err)
	}

	// Apply audit
	if err := o.Audit.ApplyTo(&config.Config); err != nil {
		return fmt.Errorf("failed to apply audit options: %w", err)
	}

	// Set version
	versionInfo := version.Get()
	config.Version = &versionInfo

	return nil
}

// applyAuthOptions applies authentication and authorization (unless disabled for testing)
func (o *Options) applyAuthOptions(config *genericapiserver.RecommendedConfig) error {
	if o.DisableAuthForTesting {
		return nil // Skip auth for testing
	}

	// Apply authentication
	if err := o.Authentication.ApplyTo(&config.Authentication, config.SecureServing, nil); err != nil {
		return fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Apply authorization
	if err := o.Authorization.ApplyTo(&config.Authorization); err != nil {
		return fmt.Errorf("failed to apply authorization: %w", err)
	}

	return nil
}

// configureOpenAPI sets up OpenAPI v2 and v3 configuration
func (o *Options) configureOpenAPI(config *genericapiserver.RecommendedConfig) {
	versionInfo := version.Get()
	namer := openapinamer.NewDefinitionNamer(server.Scheme)

	// OpenAPI v2
	config.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		generatedopenapi.GetOpenAPIDefinitions,
		namer,
	)
	config.OpenAPIConfig.Info.Title = ServerName
	config.OpenAPIConfig.Info.Version = versionInfo.GitVersion

	// OpenAPI v3
	config.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(
		generatedopenapi.GetOpenAPIDefinitions,
		namer,
	)
	config.OpenAPIV3Config.Info.Title = ServerName
	config.OpenAPIV3Config.Info.Version = versionInfo.GitVersion
}

// buildRESTConfig creates the Kubernetes client configuration
func (o *Options) buildRESTConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error

	if o.Kubeconfig != "" {
		config, err = o.loadKubeconfigFromFile()
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client config: %w", err)
	}

	// Set Kubernetes defaults
	if err := rest.SetKubernetesDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set kubernetes defaults: %w", err)
	}

	return config, nil
}

// loadKubeconfigFromFile loads kubeconfig from the specified file
func (o *Options) loadKubeconfigFromFile() (*rest.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: o.Kubeconfig,
	}
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)
	return loader.ClientConfig()
}

// buildStorageConfig creates the storage configuration
func (o *Options) buildStorageConfig() (server.StorageConfig, error) {
	config := server.StorageConfig{
		Backend:     o.StorageBackend,
		ClusterName: o.ClusterName,
	}

	switch o.StorageBackend {
	case "postgres":
		config.Postgres = o.buildPostgresConfig()
	case "etcd":
		config.Etcd = o.buildEtcdConfig()
	case "memory":
		// No additional configuration
	}

	return config, nil
}

// buildPostgresConfig creates PostgreSQL configuration
func (o *Options) buildPostgresConfig() *server.PostgresConfig {
	// Environment variables are loaded during validation
	return &server.PostgresConfig{
		Host:     o.DBHost,
		Port:     o.DBPort,
		User:     o.DBUser,
		Password: o.DBPassword,
		Database: o.DBName,
		SSLMode:  o.DBSSLMode,
	}
}

// buildEtcdConfig creates etcd configuration
func (o *Options) buildEtcdConfig() *server.EtcdConfig {
	return &server.EtcdConfig{
		Endpoints: parseEndpoints(o.EtcdEndpoints),
		TLSConfig: &rest.TLSClientConfig{
			Insecure: o.EtcdSkipTLS,
		},
	}
}

// buildServerConfig creates server configuration
func (o *Options) buildServerConfig() server.ServerConfig {
	return server.ServerConfig{
		EnablePolicyReports:    o.EnablePolicyReports,
		EnableEphemeralReports: o.EnableEphemeralReports,
		EnableOpenReports:      o.EnableOpenReports,
		Namespace:              o.ServiceNamespace,
		ServiceName:            o.ServiceName,
	}
}
