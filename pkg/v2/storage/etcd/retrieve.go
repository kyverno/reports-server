package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Get retrieves a single resource by name and namespace from etcd.
// For cluster-scoped resources, pass empty string for namespace.
//
// Returns:
//   - The resource if found
//   - NotFound error if resource doesn't exist
//   - Other errors for etcd/marshaling failures
func (r *EtcdRepository[T]) Get(ctx context.Context, filter storage.Filter) (T, error) {
	var nilObj T

	if err := filter.ValidateForGet(); err != nil {
		return nilObj, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.getKey(filter.Name, filter.Namespace)

	resp, err := r.client.Get(ctx, key)
	if err != nil {
		klog.ErrorS(err, "Failed to get resource from etcd",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return nilObj, fmt.Errorf("failed to get %s from etcd: %w", r.resourceType, err)
	}

	if len(resp.Kvs) == 0 {
		klog.V(4).InfoS("Resource not found in etcd",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"key", key,
		)
		return nilObj, errors.NewNotFound(r.gr, filter.Name)
	}

	if len(resp.Kvs) != 1 {
		klog.ErrorS(nil, "Unexpected number of resources returned from etcd",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
			"count", len(resp.Kvs),
		)
		return nilObj, fmt.Errorf("expected 1 resource, got %d", len(resp.Kvs))
	}

	// Unmarshal JSON to typed object
	var obj T
	err = json.Unmarshal(resp.Kvs[0].Value, &obj)
	if err != nil {
		klog.ErrorS(err, "Failed to unmarshal resource from etcd",
			"type", r.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
		)
		return nilObj, fmt.Errorf("failed to unmarshal %s: %w", r.resourceType, err)
	}

	klog.V(5).InfoS("Retrieved resource from etcd",
		"type", r.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
	)

	return obj, nil
}

// List retrieves all resources from etcd, optionally filtered by namespace.
// For cluster-scoped resources, namespace parameter is ignored.
// Pass empty string for namespace to get all resources across all namespaces.
//
// Returns:
//   - Slice of resources (empty slice if none found)
//   - Error for etcd/validation failures (not for empty results)
func (r *EtcdRepository[T]) List(ctx context.Context, filter storage.Filter) ([]T, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	prefix := r.getPrefix(filter.Namespace)

	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		klog.ErrorS(err, "Failed to list resources from etcd",
			"type", r.resourceType,
			"namespace", filter.Namespace,
			"prefix", prefix,
		)
		return nil, fmt.Errorf("failed to list %s from etcd: %w", r.resourceType, err)
	}

	if len(resp.Kvs) == 0 {
		klog.V(4).InfoS("No resources found in etcd",
			"type", r.resourceType,
			"namespace", filter.Namespace,
		)
		return []T{}, nil
	}

	// Unmarshal all resources
	results := make([]T, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var obj T
		err = json.Unmarshal(kv.Value, &obj)
		if err != nil {
			klog.ErrorS(err, "Failed to unmarshal resource from etcd, skipping",
				"type", r.resourceType,
				"key", string(kv.Key),
			)
			continue // Skip invalid JSON
		}
		results = append(results, obj)
	}

	klog.V(4).InfoS("Listed resources from etcd",
		"type", r.resourceType,
		"namespace", filter.Namespace,
		"count", len(results),
	)

	return results, nil
}
