package rest

import (
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceMetadata contains information about a Kubernetes resource type
type ResourceMetadata struct {
	// Kind is the resource kind (e.g., "PolicyReport")
	Kind string

	// SingularName is the singular name for the resource (e.g., "policyreport")
	SingularName string

	// ShortNames are the short names for kubectl (e.g., ["polr"])
	ShortNames []string

	// Namespaced indicates if the resource is namespace-scoped
	Namespaced bool

	// Group is the API group (e.g., "wgpolicyk8s.io")
	Group string

	// Resource is the resource name (e.g., "policyreports")
	Resource string

	// NewFunc returns a new empty instance of the resource type
	NewFunc func() runtime.Object

	// NewListFunc returns a new empty list of the resource type
	NewListFunc func() runtime.Object

	// ListItemsFunc extracts the Items slice from a list object
	// Used to populate list results
	ListItemsFunc func(list runtime.Object) []runtime.Object

	// SetListItemsFunc sets the Items slice in a list object
	// Used to build list responses
	SetListItemsFunc func(list runtime.Object, items []runtime.Object)

	// TableConverter is an optional function to convert resources to table format
	// If nil, a default converter is used
	TableConverter func(table *metav1beta1.Table, objects ...runtime.Object)
}
