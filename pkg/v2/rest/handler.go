package rest

import (
	"github.com/kyverno/reports-server/pkg/storage/api"
	v2storage "github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

// GenericRESTHandler provides a complete REST API implementation for any Kubernetes resource type.
// It uses Go generics to eliminate code duplication across resource types.
//
// Type parameters:
//   - T: The resource type (must implement both metav1.Object and runtime.Object)
//
// The handler implements all required interfaces:
//   - rest.Storage (New, Destroy)
//   - rest.Scoper (NamespaceScoped)
//   - rest.KindProvider (Kind)
//   - rest.SingularNameProvider (GetSingularName)
//   - rest.ShortNamesProvider (ShortNames)
//   - rest.StandardStorage (CRUD operations)
//   - rest.Watcher (Watch support)
//   - rest.TableConvertor (kubectl table output)
//
// Example:
//
//	handler := NewGenericRESTHandler[*v1alpha2.PolicyReport](
//	    repo,
//	    versioning,
//	    ResourceMetadata{
//	        Kind: "PolicyReport",
//	        SingularName: "policyreport",
//	        ShortNames: []string{"polr"},
//	        Namespaced: true,
//	        NewFunc: func() runtime.Object { return &v1alpha2.PolicyReport{} },
//	        NewListFunc: func() runtime.Object { return &v1alpha2.PolicyReportList{} },
//	        ListItemsFunc: extractPolicyReportItems,
//	        SetListItemsFunc: setPolicyReportItems,
//	    },
//	)
type GenericRESTHandler[T Object] struct {
	// repo is the v2 storage repository for CRUD operations
	repo v2storage.IRepository[T]

	// versioning manages resource versions for optimistic concurrency
	versioning api.Versioning

	// broadcaster handles watch events for real-time updates
	broadcaster *watch.Broadcaster

	// metadata contains resource-specific information
	metadata ResourceMetadata
}

// NewGenericRESTHandler creates a new generic REST handler
//
// Parameters:
//   - repo: v2 storage repository
//   - versioning: Resource version manager
//   - metadata: Resource metadata (kind, names, scope, etc.)
//
// Returns:
//   - A handler implementing rest.Storage and related interfaces
func NewGenericRESTHandler[T Object](
	repo v2storage.IRepository[T],
	versioning api.Versioning,
	metadata ResourceMetadata,
) IRestHandler {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &GenericRESTHandler[T]{
		repo:        repo,
		versioning:  versioning,
		broadcaster: broadcaster,
		metadata:    metadata,
	}
}

// Verify that GenericRESTHandler implements all required interfaces
// Note: These are compile-time checks, commented out because they need concrete types
// var (
// 	_ rest.Storage              = &GenericRESTHandler[Object]{}
// 	_ rest.Scoper              = &GenericRESTHandler[Object]{}
// 	_ rest.KindProvider        = &GenericRESTHandler[Object]{}
// 	_ rest.SingularNameProvider = &GenericRESTHandler[Object]{}
// 	_ rest.ShortNamesProvider  = &GenericRESTHandler[Object]{}
// 	_ rest.StandardStorage     = &GenericRESTHandler[Object]{}
// 	_ rest.Watcher            = &GenericRESTHandler[Object]{}
// 	_ rest.TableConvertor     = &GenericRESTHandler[Object]{}
// )

// New returns a new empty instance of the resource
// Implements rest.Storage
func (h *GenericRESTHandler[T]) New() runtime.Object {
	return h.metadata.NewFunc()
}

// Destroy cleans up resources (no-op for our implementation)
// Implements rest.Storage
func (h *GenericRESTHandler[T]) Destroy() {
	// No cleanup needed
}

// Kind returns the resource kind
// Implements rest.KindProvider
func (h *GenericRESTHandler[T]) Kind() string {
	return h.metadata.Kind
}

// NewList returns a new empty list of the resource type
// Implements rest.Lister
func (h *GenericRESTHandler[T]) NewList() runtime.Object {
	return h.metadata.NewListFunc()
}

// NamespaceScoped returns whether the resource is namespace-scoped
// Implements rest.Scoper
func (h *GenericRESTHandler[T]) NamespaceScoped() bool {
	return h.metadata.Namespaced
}

// GetSingularName returns the singular name of the resource
// Implements rest.SingularNameProvider
func (h *GenericRESTHandler[T]) GetSingularName() string {
	return h.metadata.SingularName
}

// ShortNames returns the short names for kubectl
// Implements rest.ShortNamesProvider
func (h *GenericRESTHandler[T]) ShortNames() []string {
	return h.metadata.ShortNames
}

// ToGroupResource returns the GroupResource for this handler
// Used for error reporting
func (m *ResourceMetadata) ToGroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    m.Group,
		Resource: m.Resource,
	}
}
