package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Update modifies an existing resource in etcd.
//
// Semantics: STRICT - Fails if resource doesn't exist (standard Kubernetes behavior)
//
// The implementation first checks if the resource exists via Get(), then performs Put.
// This ensures strict Update semantics matching Kubernetes API.
//
// Returns:
//   - nil on success
//   - storage.ErrNotFound if resource doesn't exist
//   - Other errors for etcd/marshaling failures
func (r *EtcdRepository[T]) Update(ctx context.Context, obj T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.keyFromObject(obj)

	// Check if resource exists
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		klog.ErrorS(err, "Failed to check if resource exists in etcd",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to check existence for %s: %w", r.resourceType, err)
	}

	if len(resp.Kvs) == 0 {
		klog.V(4).InfoS("Resource not found in etcd, cannot update",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"key", key,
		)
		return errors.NewNotFound(r.gr, obj.GetName())
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(obj)
	if err != nil {
		klog.ErrorS(err, "Failed to marshal resource",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to marshal %s: %w", r.resourceType, err)
	}

	// Update in etcd
	_, err = r.client.Put(ctx, key, string(jsonData))
	if err != nil {
		klog.ErrorS(err, "Failed to update resource in etcd",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to update %s in etcd: %w", r.resourceType, err)
	}

	klog.V(4).InfoS("Updated resource in etcd",
		"type", r.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"key", key,
	)

	return nil
}
