package etcd

import (
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EtcdRepository provides CRUD operations for any Kubernetes resource type using etcd.
// It uses Go generics to provide type-safe operations while eliminating code duplication.
//
// Type parameter T must implement metav1.Object interface (GetName, GetNamespace, etc.)
//
// Thread-safety: This repository is safe for concurrent use via sync.Mutex.
// All operations use a simple Mutex (not RWMutex) because etcd client handles
// concurrent operations internally and we need exclusive access for consistency checks.
//
// Implements: storage.IRepository[T]
type EtcdRepository[T metav1.Object] struct {
	// mu protects etcd operations for consistency
	mu sync.Mutex

	// client is the etcd KV client for storage operations
	client clientv3.KV

	// gvk is the GroupVersionKind for this resource type
	gvk schema.GroupVersionKind

	// gr is the GroupResource for Kubernetes API errors
	gr schema.GroupResource

	// namespaced indicates if this resource is namespace-scoped
	namespaced bool

	// resourceType is used for logging and error messages (e.g., "PolicyReport")
	resourceType string
}

// NewEtcdRepository creates a new etcd repository instance.
//
// Parameters:
//   - client: etcd KV client for storage operations
//   - gvk: GroupVersionKind for the resource type
//   - gr: GroupResource for Kubernetes API compatibility
//   - resourceType: Human-readable resource type for logging/errors
//   - namespaced: Whether the resource is namespace-scoped
//
// Returns:
//   - storage.IRepository[T]: A storage-agnostic repository implementation
//
// Example:
//
//	client, _ := clientv3.New(clientv3.Config{...})
//	repo := NewEtcdRepository[*v1alpha2.PolicyReport](
//	    client,
//	    schema.GroupVersionKind{Group: "wgpolicyk8s.io", Version: "v1alpha2", Kind: "PolicyReport"},
//	    schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "policyreports"},
//	    "PolicyReport",
//	    true,
//	)
func NewEtcdRepository[T metav1.Object](
	client clientv3.KV,
	gvk schema.GroupVersionKind,
	gr schema.GroupResource,
	resourceType string,
	namespaced bool,
) storage.IRepository[T] {
	return &EtcdRepository[T]{
		client:       client,
		gvk:          gvk,
		gr:           gr,
		resourceType: resourceType,
		namespaced:   namespaced,
	}
}

// getPrefix returns the etcd key prefix for listing operations.
// For namespaced resources with namespace: "group/version/kind/namespace/"
// For namespaced resources without namespace: "group/version/kind/"
// For cluster-scoped resources: "group/version/kind/"
func (r *EtcdRepository[T]) getPrefix(namespace string) string {
	if r.namespaced && namespace != "" {
		return fmt.Sprintf("%s/%s/%s/%s/", r.gvk.Group, r.gvk.Version, r.gvk.Kind, namespace)
	}
	return fmt.Sprintf("%s/%s/%s/", r.gvk.Group, r.gvk.Version, r.gvk.Kind)
}

// getKey generates the full etcd key for a specific resource.
// Format: "group/version/kind/namespace/name" for namespaced
// Format: "group/version/kind/name" for cluster-scoped
func (r *EtcdRepository[T]) getKey(name, namespace string) string {
	return fmt.Sprintf("%s%s", r.getPrefix(namespace), name)
}

// keyFromObject generates the etcd key from a resource object.
func (r *EtcdRepository[T]) keyFromObject(obj T) string {
	return r.getKey(obj.GetName(), obj.GetNamespace())
}
