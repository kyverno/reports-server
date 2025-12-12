package storage

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IRepository defines generic CRUD operations for any Kubernetes resource type.
// This interface is storage-agnostic and can be implemented by:
//   - PostgreSQL (SQL-based storage)
//   - etcd (key-value storage)
//   - In-memory (map-based storage)
//   - Any future storage backend
//
// Type parameter T must implement metav1.Object (GetName, GetNamespace, etc.)
//
// All query operations use the Filter struct for consistent API.
type IRepository[T metav1.Object] interface {
	// Get retrieves a single resource by filter.
	// Filter should specify Name (required) and Namespace (for namespaced resources).
	//
	// Example:
	//   filter := NewFilter("my-report", "default")
	//   report, err := repo.Get(ctx, filter)
	//
	// Returns:
	//   - The resource if found
	//   - NotFound error if resource doesn't exist
	//   - Other errors for storage failures
	Get(ctx context.Context, filter Filter) (T, error)

	// List retrieves resources matching the filter criteria.
	//
	// Filter examples:
	//   // List all in namespace
	//   filter := Filter{Namespace: "default"}
	//
	//   // List all across namespaces
	//   filter := Filter{}
	//
	// Returns:
	//   - Slice of resources (empty slice if none found)
	//   - Error for storage failures
	List(ctx context.Context, filter Filter) ([]T, error)

	// Create inserts a new resource.
	//
	// Semantics: STRICT - Fails if resource already exists
	// This matches standard Kubernetes API semantics.
	//
	// The implementation checks if resource exists first via Get(),
	// then performs INSERT (without ON CONFLICT).
	//
	// Returns:
	//   - nil on success (resource created)
	//   - storage.ErrAlreadyExists if resource already exists
	//   - Other errors for database/marshaling failures
	Create(ctx context.Context, obj T) error

	// Update modifies an existing resource.
	//
	// Semantics: STRICT - Fails if resource doesn't exist
	// This matches standard Kubernetes API semantics.
	//
	// The implementation performs plain UPDATE and checks rowsAffected.
	// If rowsAffected=0, returns ErrNotFound.
	//
	// Returns:
	//   - nil on success (resource updated)
	//   - storage.ErrNotFound if resource doesn't exist
	//   - Other errors for database/marshaling failures
	Update(ctx context.Context, obj T) error

	// Delete removes a resource matching the filter.
	// Filter should specify Name (required) and Namespace (for namespaced resources).
	//
	// Example:
	//   filter := NewFilter("my-report", "default")
	//   err := repo.Delete(ctx, filter)
	//
	// Returns:
	//   - NotFound error if resource doesn't exist
	//   - Other errors for storage failures
	Delete(ctx context.Context, filter Filter) error
}
