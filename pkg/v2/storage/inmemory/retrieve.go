package inmemory

import (
	"context"
	"strings"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Get retrieves a single resource by name and namespace.
// For cluster-scoped resources, pass empty string for namespace.
//
// Returns:
//   - The resource if found
//   - NotFound error if resource doesn't exist
//   - Other errors for validation failures
func (r *InMemoryRepository[T]) Get(ctx context.Context, filter storage.Filter) (T, error) {
	var nilObj T

	if err := filter.ValidateForGet(); err != nil {
		return nilObj, err
	}

	key := r.key(filter.Name, filter.Namespace)

	r.mu.RLock()
	obj, found := r.db[key]
	r.mu.RUnlock()

	if !found {
		klog.V(4).InfoS("Resource not found",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return nilObj, errors.NewNotFound(r.gr, filter.Name)
	}

	klog.V(5).InfoS("Retrieved resource",
		"type", r.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
	)

	return obj, nil
}

// List retrieves all resources, optionally filtered by namespace.
// For cluster-scoped resources, namespace parameter is ignored.
// Pass empty string for namespace to get all resources across all namespaces.
//
// Returns:
//   - Slice of resources (empty slice if none found)
//   - Error only for validation failures (not for empty results)
func (r *InMemoryRepository[T]) List(ctx context.Context, filter storage.Filter) ([]T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]T, 0, len(r.db))

	// If namespace filter is provided, filter by namespace
	if filter.Namespace != "" && r.namespaced {
		prefix := filter.Namespace + "/"
		for key, obj := range r.db {
			if strings.HasPrefix(key, prefix) {
				results = append(results, obj)
			}
		}
	} else {
		// No namespace filter - return all resources
		for _, obj := range r.db {
			results = append(results, obj)
		}
	}

	klog.V(4).InfoS("Listed resources",
		"type", r.resourceType,
		"namespace", filter.Namespace,
		"count", len(results),
	)

	return results, nil
}
