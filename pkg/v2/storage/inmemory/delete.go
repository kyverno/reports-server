package inmemory

import (
	"context"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Delete removes a resource matching the filter.
// Filter should specify Name (required) and Namespace (for namespaced resources).
//
// Returns:
//   - nil on success
//   - NotFound error if resource doesn't exist
//   - Other errors for validation failures
func (r *InMemoryRepository[T]) Delete(ctx context.Context, filter storage.Filter) error {
	if err := filter.ValidateForDelete(); err != nil {
		return err
	}

	key := r.key(filter.Name, filter.Namespace)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	_, found := r.db[key]
	if !found {
		klog.V(4).InfoS("Resource not found, cannot delete",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return errors.NewNotFound(r.gr, filter.Name)
	}

	// Delete the resource
	delete(r.db, key)

	klog.V(4).InfoS("Deleted resource",
		"type", r.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
		"key", key,
	)

	return nil
}
