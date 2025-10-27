package server

import (
	"database/sql"
	"fmt"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	pgconfig "github.com/kyverno/reports-server/pkg/v2/config/postgres"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"github.com/kyverno/reports-server/pkg/v2/storage/etcd"
	"github.com/kyverno/reports-server/pkg/v2/storage/inmemory"
	"github.com/kyverno/reports-server/pkg/v2/storage/postgres"
	"github.com/kyverno/reports-server/pkg/v2/versioning"
	_ "github.com/lib/pq"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

// API Group constants
const (
	GroupWGPolicy       = "wgpolicyk8s.io"
	GroupKyvernoReports = "reports.kyverno.io"
	GroupOpenReports    = "openreports.io"
)

// Config holds the configuration for the v2 reports server
type Config struct {
	// GenericConfig is the base API server configuration
	GenericConfig *genericapiserver.RecommendedConfig

	// RESTConfig for communicating with the Kubernetes API
	RESTConfig *rest.Config

	// Storage configuration
	Storage StorageConfig

	// Server configuration
	Server ServerConfig

	// Versioning manages resource versions
	Versioning versioning.Versioning
}

// StorageConfig configures the storage backend
type StorageConfig struct {
	// Backend specifies which storage to use: "postgres", "etcd", or "memory"
	Backend string

	// PostgresConfig for Postgres storage
	Postgres *PostgresConfig

	// EtcdConfig for etcd storage
	Etcd *EtcdConfig

	// ClusterID uniquely identifies this cluster
	ClusterID string

	// ClusterName is the human-readable cluster name
	ClusterName string
}

// PostgresConfig holds Postgres connection settings
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// EtcdConfig holds etcd connection settings
type EtcdConfig struct {
	Endpoints []string
	TLSConfig *rest.TLSClientConfig
}

// ServerConfig holds server-specific settings
type ServerConfig struct {
	// EnablePolicyReports enables wgpolicyk8s.io API group
	EnablePolicyReports bool

	// EnableEphemeralReports enables reports.kyverno.io API group
	EnableEphemeralReports bool

	// EnableOpenReports enables openreports.io API group
	EnableOpenReports bool

	// Namespace is the namespace where the server runs
	Namespace string

	// ServiceName is the name of the service
	ServiceName string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Storage: StorageConfig{
			Backend: "memory", // Safe default for dev/testing
		},
		Server: ServerConfig{
			EnablePolicyReports:    true,
			EnableEphemeralReports: true,
			EnableOpenReports:      true,
			Namespace:              "reports-server",
			ServiceName:            "reports-server",
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.GenericConfig == nil {
		return ErrMissingGenericConfig
	}

	if c.RESTConfig == nil {
		return ErrMissingRESTConfig
	}

	if c.Versioning == nil {
		return ErrMissingVersioning
	}

	return c.Storage.Validate()
}

// Validate checks if storage configuration is valid
func (s *StorageConfig) Validate() error {
	switch s.Backend {
	case "postgres":
		if s.Postgres == nil {
			return ErrMissingPostgresConfig
		}
		return s.Postgres.Validate()
	case "etcd":
		if s.Etcd == nil {
			return ErrMissingEtcdConfig
		}
		return s.Etcd.Validate()
	case "memory":
		return nil
	default:
		return ErrInvalidStorageBackend
	}
}

// Validate checks if Postgres configuration is valid
func (p *PostgresConfig) Validate() error {
	if p.Host == "" {
		return ErrMissingPostgresHost
	}
	if p.Database == "" {
		return ErrMissingPostgresDatabase
	}
	if p.User == "" {
		return ErrMissingPostgresUser
	}
	return nil
}

// Validate checks if etcd configuration is valid
func (e *EtcdConfig) Validate() error {
	if len(e.Endpoints) == 0 {
		return ErrMissingEtcdEndpoints
	}
	return nil
}

// Complete fills in missing fields with defaults and creates the server
func (c *Config) Complete() (*Server, error) {
	// Validate configuration
	if err := c.Validate(); err != nil {
		return nil, err
	}

	// Get or generate cluster ID
	if c.Storage.ClusterID == "" {
		c.Storage.ClusterID = GetClusterIDOrGenerate(c.RESTConfig, c.Storage.ClusterName)
		klog.InfoS("Using cluster ID", "clusterID", c.Storage.ClusterID)
	}

	// Setup metrics handler
	c.GenericConfig.EnableMetrics = false // Disable default metrics
	metricsHandler, err := metrics.CreateHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics handler: %w", err)
	}

	// Create generic API server
	genericServer, err := c.GenericConfig.Complete().New(
		"reports-server-v2",
		genericapiserver.NewEmptyDelegate(),
	)
	if err != nil {
		return nil, err
	}

	// Register custom metrics endpoint
	genericServer.Handler.NonGoRestfulMux.HandleFunc("/metrics", metricsHandler)

	// Create storage repositories based on configuration
	repositories, err := c.createRepositories()
	if err != nil {
		return nil, err
	}

	// Install APIServices in Kubernetes
	if err := c.InstallAPIServices(); err != nil {
		klog.ErrorS(err, "Failed to install APIServices (non-fatal, continuing)")
		// Non-fatal - server can still work without APIServices registered
	}

	// Create and return the server
	return &Server{
		GenericAPIServer: genericServer,
		repositories:     repositories,
		config:           c,
	}, nil
}

// createRepositories creates storage repositories based on configuration
func (c *Config) createRepositories() (*Repositories, error) {
	switch c.Storage.Backend {
	case "postgres":
		return c.createPostgresRepositories()
	case "etcd":
		return c.createEtcdRepositories()
	case "memory":
		return c.createInMemoryRepositories()
	default:
		return nil, ErrInvalidStorageBackend
	}
}

// createPostgresRepositories creates Postgres-backed repositories
func (c *Config) createPostgresRepositories() (*Repositories, error) {
	klog.Info("Creating Postgres repositories")

	// Create Postgres config
	hostCfg := pgconfig.NewHostConfig(c.Storage.Postgres.Host, nil)
	sslCfg := pgconfig.NewSSLConfig(c.Storage.Postgres.SSLMode, "", "", "")
	pgCfg := pgconfig.NewConfig(
		hostCfg,
		c.Storage.Postgres.Port,
		c.Storage.Postgres.User,
		c.Storage.Postgres.Password,
		c.Storage.Postgres.Database,
		sslCfg,
	)

	// Connect to database
	primaryDB, err := sql.Open("postgres", pgCfg.PrimaryDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test connection
	if err := primaryDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	klog.InfoS("Connected to Postgres", "host", c.Storage.Postgres.Host, "database", c.Storage.Postgres.Database)

	// Create router (no read replicas for now)
	router := postgres.NewDBRouter(primaryDB, nil)

	// Create repositories for all resource types
	return &Repositories{
		PolicyReports: postgres.NewPostgresRepository[*v1alpha2.PolicyReport](
			router,
			c.Storage.ClusterID,
			"policyreports",
			"PolicyReport",
			true,
			schema.GroupResource{Group: GroupWGPolicy, Resource: "policyreports"},
		),
		ClusterPolicyReports: postgres.NewPostgresRepository[*v1alpha2.ClusterPolicyReport](
			router,
			c.Storage.ClusterID,
			"clusterpolicyreports",
			"ClusterPolicyReport",
			false,
			schema.GroupResource{Group: GroupWGPolicy, Resource: "clusterpolicyreports"},
		),
		EphemeralReports: postgres.NewPostgresRepository[*reportsv1.EphemeralReport](
			router,
			c.Storage.ClusterID,
			"ephemeralreports",
			"EphemeralReport",
			true,
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "ephemeralreports"},
		),
		ClusterEphemeralReports: postgres.NewPostgresRepository[*reportsv1.ClusterEphemeralReport](
			router,
			c.Storage.ClusterID,
			"clusterephemeralreports",
			"ClusterEphemeralReport",
			false,
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "clusterephemeralreports"},
		),
		Reports: postgres.NewPostgresRepository[*openreportsv1alpha1.Report](
			router,
			c.Storage.ClusterID,
			"reports",
			"Report",
			true,
			schema.GroupResource{Group: GroupOpenReports, Resource: "reports"},
		),
		ClusterReports: postgres.NewPostgresRepository[*openreportsv1alpha1.ClusterReport](
			router,
			c.Storage.ClusterID,
			"clusterreports",
			"ClusterReport",
			false,
			schema.GroupResource{Group: GroupOpenReports, Resource: "clusterreports"},
		),
	}, nil
}

// createEtcdRepositories creates etcd-backed repositories
func (c *Config) createEtcdRepositories() (*Repositories, error) {
	klog.Info("Creating etcd repositories")

	// Create etcd client
	etcdCfg := clientv3.Config{
		Endpoints: c.Storage.Etcd.Endpoints,
	}

	// Convert TLS config if provided
	if c.Storage.Etcd.TLSConfig != nil {
		restCfg := &rest.Config{
			TLSClientConfig: *c.Storage.Etcd.TLSConfig,
		}
		tlsCfg, err := rest.TLSConfigFor(restCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		etcdCfg.TLS = tlsCfg
	}

	client, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	klog.InfoS("Connected to etcd", "endpoints", c.Storage.Etcd.Endpoints)

	// Create repositories for all resource types
	return &Repositories{
		PolicyReports: etcd.NewEtcdRepository[*v1alpha2.PolicyReport](
			client,
			schema.GroupVersionKind{Group: GroupWGPolicy, Version: "v1alpha2", Kind: "PolicyReport"},
			schema.GroupResource{Group: GroupWGPolicy, Resource: "policyreports"},
			"PolicyReport",
			true,
		),
		ClusterPolicyReports: etcd.NewEtcdRepository[*v1alpha2.ClusterPolicyReport](
			client,
			schema.GroupVersionKind{Group: GroupWGPolicy, Version: "v1alpha2", Kind: "ClusterPolicyReport"},
			schema.GroupResource{Group: GroupWGPolicy, Resource: "clusterpolicyreports"},
			"ClusterPolicyReport",
			false,
		),
		EphemeralReports: etcd.NewEtcdRepository[*reportsv1.EphemeralReport](
			client,
			schema.GroupVersionKind{Group: GroupKyvernoReports, Version: "v1", Kind: "EphemeralReport"},
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "ephemeralreports"},
			"EphemeralReport",
			true,
		),
		ClusterEphemeralReports: etcd.NewEtcdRepository[*reportsv1.ClusterEphemeralReport](
			client,
			schema.GroupVersionKind{Group: GroupKyvernoReports, Version: "v1", Kind: "ClusterEphemeralReport"},
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "clusterephemeralreports"},
			"ClusterEphemeralReport",
			false,
		),
		Reports: etcd.NewEtcdRepository[*openreportsv1alpha1.Report](
			client,
			schema.GroupVersionKind{Group: GroupOpenReports, Version: "v1alpha1", Kind: "Report"},
			schema.GroupResource{Group: GroupOpenReports, Resource: "reports"},
			"Report",
			true,
		),
		ClusterReports: etcd.NewEtcdRepository[*openreportsv1alpha1.ClusterReport](
			client,
			schema.GroupVersionKind{Group: GroupOpenReports, Version: "v1alpha1", Kind: "ClusterReport"},
			schema.GroupResource{Group: GroupOpenReports, Resource: "clusterreports"},
			"ClusterReport",
			false,
		),
	}, nil
}

// createInMemoryRepositories creates in-memory repositories
func (c *Config) createInMemoryRepositories() (*Repositories, error) {
	klog.Info("Creating in-memory repositories")

	return &Repositories{
		PolicyReports: inmemory.NewInMemoryRepository[*v1alpha2.PolicyReport](
			"PolicyReport",
			true,
			schema.GroupResource{Group: GroupWGPolicy, Resource: "policyreports"},
		),
		ClusterPolicyReports: inmemory.NewInMemoryRepository[*v1alpha2.ClusterPolicyReport](
			"ClusterPolicyReport",
			false,
			schema.GroupResource{Group: GroupWGPolicy, Resource: "clusterpolicyreports"},
		),
		EphemeralReports: inmemory.NewInMemoryRepository[*reportsv1.EphemeralReport](
			"EphemeralReport",
			true,
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "ephemeralreports"},
		),
		ClusterEphemeralReports: inmemory.NewInMemoryRepository[*reportsv1.ClusterEphemeralReport](
			"ClusterEphemeralReport",
			false,
			schema.GroupResource{Group: GroupKyvernoReports, Resource: "clusterephemeralreports"},
		),
		Reports: inmemory.NewInMemoryRepository[*openreportsv1alpha1.Report](
			"Report",
			true,
			schema.GroupResource{Group: GroupOpenReports, Resource: "reports"},
		),
		ClusterReports: inmemory.NewInMemoryRepository[*openreportsv1alpha1.ClusterReport](
			"ClusterReport",
			false,
			schema.GroupResource{Group: GroupOpenReports, Resource: "clusterreports"},
		),
	}, nil
}

// Repositories holds all storage repositories
type Repositories struct {
	// PolicyReports repository
	PolicyReports storage.IRepository[*v1alpha2.PolicyReport]

	// ClusterPolicyReports repository
	ClusterPolicyReports storage.IRepository[*v1alpha2.ClusterPolicyReport]

	// EphemeralReports repository
	EphemeralReports storage.IRepository[*reportsv1.EphemeralReport]

	// ClusterEphemeralReports repository
	ClusterEphemeralReports storage.IRepository[*reportsv1.ClusterEphemeralReport]

	// Reports (openreports.io) repository
	Reports storage.IRepository[*openreportsv1alpha1.Report]

	// ClusterReports (openreports.io) repository
	ClusterReports storage.IRepository[*openreportsv1alpha1.ClusterReport]
}
