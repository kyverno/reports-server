package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/klog/v2"
)

// Create inserts a new resource into etcd.
//
// Semantics: STRICT - Fails if resource already exists (standard Kubernetes behavior)
//
// The implementation first checks if the resource exists via Get(), then performs Put.
// This ensures strict Create semantics matching Kubernetes API.
//
// Returns:
//   - nil on success
//   - storage.ErrAlreadyExists if resource already exists
//   - Other errors for etcd/marshaling failures
func (r *EtcdRepository[T]) Create(ctx context.Context, obj T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.keyFromObject(obj)

	// Check if resource already exists
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		klog.ErrorS(err, "Failed to check if resource exists in etcd",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to check existence for %s: %w", r.resourceType, err)
	}

	if len(resp.Kvs) > 0 {
		klog.V(4).InfoS("Resource already exists in etcd, cannot create",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"key", key,
		)
		return storage.NewAlreadyExistsError(r.resourceType, obj.GetName(), obj.GetNamespace())
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

	// Store in etcd
	_, err = r.client.Put(ctx, key, string(jsonData))
	if err != nil {
		klog.ErrorS(err, "Failed to create resource in etcd",
			"type", r.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to create %s in etcd: %w", r.resourceType, err)
	}

	klog.V(4).InfoS("Created resource in etcd",
		"type", r.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"key", key,
	)

	return nil
}
