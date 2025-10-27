package postgres

import (
	"github.com/kyverno/reports-server/pkg/v2/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PostgresRepository provides CRUD operations for any Kubernetes resource type using PostgreSQL.
// It uses Go generics to provide type-safe operations while eliminating code duplication.
//
// Type parameter T must implement metav1.Object interface (GetName, GetNamespace, etc.)
//
// Thread-safety: This repository is safe for concurrent use. It does NOT use mutexes
// because the underlying database connection pool is already thread-safe.
//
// Implements: storage.IRepository[T]
type PostgresRepository[T metav1.Object] struct {
	// router handles read/write splitting between primary and replicas
	router *DBRouter

	// clusterID uniquely identifies this Kubernetes cluster
	clusterID string

	// tableName is the PostgreSQL table name (e.g., "policyreports")
	tableName string

	// namespaced indicates if this resource is namespace-scoped
	namespaced bool

	// resourceType is used for logging and metrics (e.g., "PolicyReport")
	resourceType string

	// gr is the GroupResource for Kubernetes API errors
	gr schema.GroupResource
}

// NewPostgresRepository creates a new PostgreSQL repository instance.
//
// Parameters:
//   - router: Database router for primary/replica management
//   - clusterID: Unique identifier for the Kubernetes cluster
//   - tableName: PostgreSQL table name
//   - resourceType: Human-readable resource type for logging/metrics
//   - namespaced: Whether the resource is namespace-scoped
//   - gr: GroupResource for Kubernetes API compatibility
//
// Returns:
//   - storage.IRepository[T]: A storage-agnostic repository implementation
//
// Example:
//
//	repo := NewPostgresRepository[*v1alpha2.PolicyReport](
//	    router,
//	    clusterID,
//	    "policyreports",
//	    "PolicyReport",
//	    true,
//	    schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "policyreports"},
//	)
func NewPostgresRepository[T metav1.Object](
	router *DBRouter,
	clusterID string,
	tableName string,
	resourceType string,
	namespaced bool,
	gr schema.GroupResource,
) storage.IRepository[T] {
	return &PostgresRepository[T]{
		router:       router,
		clusterID:    clusterID,
		tableName:    tableName,
		namespaced:   namespaced,
		resourceType: resourceType,
		gr:           gr,
	}
}
