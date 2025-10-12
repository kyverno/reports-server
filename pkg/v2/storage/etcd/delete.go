package etcd

import (
	"context"
	"fmt"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Delete removes a resource from etcd matching the filter.
// Filter should specify Name (required) and Namespace (for namespaced resources).
//
// Returns:
//   - nil on success
//   - NotFound error if resource doesn't exist
//   - Other errors for etcd/validation failures
func (r *EtcdRepository[T]) Delete(ctx context.Context, filter storage.Filter) error {
	if err := filter.ValidateForDelete(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.getKey(filter.Name, filter.Namespace)

	// Delete from etcd
	resp, err := r.client.Delete(ctx, key)
	if err != nil {
		klog.ErrorS(err, "Failed to delete resource from etcd",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return fmt.Errorf("failed to delete %s from etcd: %w", r.resourceType, err)
	}

	// Check if resource was actually deleted
	if resp.Deleted == 0 {
		klog.V(4).InfoS("Resource not found in etcd, cannot delete",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return errors.NewNotFound(r.gr, filter.Name)
	}

	klog.V(4).InfoS("Deleted resource from etcd",
		"type", r.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
		"key", key,
		"deleted", resp.Deleted,
	)

	return nil
}
