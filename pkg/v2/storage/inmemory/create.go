package inmemory

import (
	"context"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/klog/v2"
)

// Create inserts a new resource.
//
// Semantics: STRICT - Fails if resource already exists (standard Kubernetes behavior)
//
// Returns:
//   - nil on success
//   - storage.ErrAlreadyExists if resource already exists
//   - Other errors for validation failures
func (r *InMemoryRepository[T]) Create(ctx context.Context, obj T) error {
	key := r.keyFromObject(obj)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource already exists
	if _, found := r.db[key]; found {
		klog.V(4).InfoS("Resource already exists, cannot create",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"key", key,
		)
		return storage.NewAlreadyExistsError(r.resourceType, obj.GetName(), obj.GetNamespace())
	}

	// Store the resource
	r.db[key] = obj

	klog.V(4).InfoS("Created resource",
		"type", r.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"key", key,
	)

	return nil
}
