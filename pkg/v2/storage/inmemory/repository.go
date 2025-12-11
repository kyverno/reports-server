package inmemory

import (
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// InMemoryRepository provides CRUD operations for any Kubernetes resource type using in-memory storage.
// It uses Go generics to provide type-safe operations while eliminating code duplication.
//
// Type parameter T must implement metav1.Object interface (GetName, GetNamespace, etc.)
//
// Thread-safety: This repository is safe for concurrent use via sync.RWMutex.
// Read operations (Get, List) use RLock for concurrent reads.
// Write operations (Create, Update, Delete) use Lock for exclusive access.
//
// Implements: storage.IRepository[T]
type InMemoryRepository[T metav1.Object] struct {
	// mu protects the db map for thread-safe operations
	mu sync.RWMutex

	// db stores resources keyed by their unique identifier
	// For namespaced resources: "namespace/name"
	// For cluster-scoped resources: "name"
	db map[string]T

	// namespaced indicates if this resource is namespace-scoped
	namespaced bool

	// resourceType is used for logging and error messages (e.g., "PolicyReport")
	resourceType string

	// gr is the GroupResource for Kubernetes API errors
	gr schema.GroupResource
}

// NewInMemoryRepository creates a new in-memory repository instance.
//
// Parameters:
//   - resourceType: Human-readable resource type for logging/errors
//   - namespaced: Whether the resource is namespace-scoped
//   - gr: GroupResource for Kubernetes API compatibility
//
// Returns:
//   - storage.IRepository[T]: A storage-agnostic repository implementation
//
// Example:
//
//	repo := NewInMemoryRepository[*v1alpha2.PolicyReport](
//	    "PolicyReport",
//	    true,
//	    schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "policyreports"},
//	)
func NewInMemoryRepository[T metav1.Object](
	resourceType string,
	namespaced bool,
	gr schema.GroupResource,
) storage.IRepository[T] {
	return &InMemoryRepository[T]{
		db:           make(map[string]T),
		namespaced:   namespaced,
		resourceType: resourceType,
		gr:           gr,
	}
}

// key generates a unique key for a resource.
// For namespaced resources: "namespace/name"
// For cluster-scoped resources: "name"
func (r *InMemoryRepository[T]) key(name, namespace string) string {
	if r.namespaced && namespace != "" {
		return fmt.Sprintf("%s/%s", namespace, name)
	}
	return name
}

// keyFromObject generates a unique key from a resource object.
func (r *InMemoryRepository[T]) keyFromObject(obj T) string {
	return r.key(obj.GetName(), obj.GetNamespace())
}
