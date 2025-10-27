package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/v2/config/postgres"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

const (
	maxRetries      = 10
	sleepDuration   = 15 * time.Second
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
)

type Storage struct {
	Router        *DBRouter
	PolrRepo      storage.IRepository[*v1alpha2.PolicyReport]
	CpolrRepo     storage.IRepository[*v1alpha2.ClusterPolicyReport]
	EphrRepo      storage.IRepository[*reportsv1.EphemeralReport]
	CephrRepo     storage.IRepository[*reportsv1.ClusterEphemeralReport]
	OrRepo        storage.IRepository[*openreportsv1alpha1.Report]
	OrClusterRepo storage.IRepository[*openreportsv1alpha1.ClusterReport]
}

// New creates a new PostgreSQL storage implementation using generic repositories.
//
// This is the V2 implementation that:
//   - Uses Go generics to eliminate code duplication
//   - Removes all mutexes (database handles concurrency)
//   - Provides proper read/write splitting with replicas
//   - Has better observability and error handling
//
// Parameters:
//   - config: PostgreSQL connection configuration
//   - clusterID: Unique identifier for this Kubernetes cluster
//
// Returns:
//   - storage.IRepository: Storage interface implementation
//   - error: Connection or initialization error
func New(config *postgres.Config, clusterID string) (*Storage, error) {
	klog.InfoS("Initializing PostgreSQL storage v2",
		"host", config.HostConfig.PrimaryHost,
		"database", config.DBname,
		"replicas", len(config.HostConfig.ReadReplicaHosts),
	)

	// Validate configuration before connecting
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Connect to primary database
	primaryDB, err := connectAndConfigure(config.PrimaryDSN(), "primary")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary: %w", err)
	}

	// Connect to read replicas
	var readReplicas []*sql.DB
	replicaDSNs := config.ReadReplicaDSNs()
	for i, dsn := range replicaDSNs {
		replicaDB, err := connectAndConfigure(dsn, fmt.Sprintf("replica-%d", i))
		if err != nil {
			klog.ErrorS(err, "Failed to connect to read replica, skipping",
				"dsn", dsn,
				"index", i,
			)
			continue // Skip failed replicas, don't fail startup
		}

		readReplicas = append(readReplicas, replicaDB)
	}

	// Initialize database schema (tables and indexes)
	if err := InitializeSchema(context.Background(), primaryDB); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Create database router
	router := NewDBRouter(primaryDB, readReplicas)

	// Create PostgreSQL repositories for each resource type
	polrRepo := NewPostgresRepository[*v1alpha2.PolicyReport](
		router,
		clusterID,
		"policyreports",
		"PolicyReport",
		true, // namespaced
		schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "policyreports"},
	)

	cpolrRepo := NewPostgresRepository[*v1alpha2.ClusterPolicyReport](
		router,
		clusterID,
		"clusterpolicyreports",
		"ClusterPolicyReport",
		false, // cluster-scoped
		schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "clusterpolicyreports"},
	)

	ephrRepo := NewPostgresRepository[*reportsv1.EphemeralReport](
		router,
		clusterID,
		"ephemeralreports",
		"EphemeralReport",
		true, // namespaced
		schema.GroupResource{Group: "reports.kyverno.io", Resource: "ephemeralreports"},
	)

	cephrRepo := NewPostgresRepository[*reportsv1.ClusterEphemeralReport](
		router,
		clusterID,
		"clusterephemeralreports",
		"ClusterEphemeralReport",
		false, // cluster-scoped
		schema.GroupResource{Group: "reports.kyverno.io", Resource: "clusterephemeralreports"},
	)

	orRepo := NewPostgresRepository[*openreportsv1alpha1.Report](
		router,
		clusterID,
		"reports",
		"Report",
		true, // namespaced
		schema.GroupResource{Group: "openreports.io", Resource: "reports"},
	)

	orClusterRepo := NewPostgresRepository[*openreportsv1alpha1.ClusterReport](
		router,
		clusterID,
		"clusterreports",
		"ClusterReport",
		false, // cluster-scoped
		schema.GroupResource{Group: "openreports.io", Resource: "clusterreports"},
	)

	klog.InfoS("Successfully initialized PostgreSQL storage v2",
		"clusterID", clusterID,
		"primaryConnected", true,
		"replicasConnected", len(readReplicas),
	)

	return &Storage{
		Router:        router,
		PolrRepo:      polrRepo,
		CpolrRepo:     cpolrRepo,
		EphrRepo:      ephrRepo,
		CephrRepo:     cephrRepo,
		OrRepo:        orRepo,
		OrClusterRepo: orClusterRepo,
	}, nil
}

// connectAndConfigure opens a database connection and configures the connection pool
func connectAndConfigure(dsn string, name string) (*sql.DB, error) {
	klog.InfoS("Connecting to database", "name", name)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection with retries
	if err := pingWithRetry(db, name); err != nil {
		db.Close()
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	klog.InfoS("Database connection configured",
		"name", name,
		"maxOpenConns", maxOpenConns,
		"maxIdleConns", maxIdleConns,
		"connMaxLifetime", connMaxLifetime,
		"connMaxIdleTime", connMaxIdleTime,
	)

	return db, nil
}

// pingWithRetry attempts to ping the database with exponential backoff
func pingWithRetry(db *sql.DB, name string) error {
	ctx := context.Background()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		klog.V(4).InfoS("Pinging database",
			"name", name,
			"attempt", attempt,
			"maxRetries", maxRetries,
		)

		if err := db.PingContext(ctx); err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
			}

			klog.V(4).InfoS("Ping failed, retrying",
				"name", name,
				"attempt", attempt,
				"error", err,
				"sleepDuration", sleepDuration,
			)

			time.Sleep(sleepDuration)
			continue
		}

		klog.InfoS("Successfully connected to database",
			"name", name,
			"attempts", attempt,
		)
		return nil
	}

	return fmt.Errorf("failed to connect after %d attempts", maxRetries)
}
