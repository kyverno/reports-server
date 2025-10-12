package inmemory

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Update modifies an existing resource.
//
// Semantics: STRICT - Fails if resource doesn't exist (standard Kubernetes behavior)
//
// Returns:
//   - nil on success
//   - storage.ErrNotFound if resource doesn't exist
//   - Other errors for validation failures
func (r *InMemoryRepository[T]) Update(ctx context.Context, obj T) error {
	key := r.keyFromObject(obj)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	if _, found := r.db[key]; !found {
		klog.V(4).InfoS("Resource not found, cannot update",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"key", key,
		)
		return errors.NewNotFound(r.gr, obj.GetName())
	}

	// Update the resource
	r.db[key] = obj

	klog.V(4).InfoS("Updated resource",
		"type", r.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"key", key,
	)

	return nil
}
