package rest

import (
	"context"
	"fmt"
	"strconv"

	v2storage "github.com/kyverno/reports-server/pkg/v2/storage"
	errorpkg "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// Get retrieves a single resource by name
// Implements rest.Getter
func (h *GenericRESTHandler[T]) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.V(4).InfoS("Getting resource",
		"kind", h.metadata.Kind,
		"name", name,
		"namespace", namespace,
	)

	filter := v2storage.NewFilter(name, namespace)
	obj, err := h.repo.Get(ctx, filter)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(4).InfoS("Resource not found",
				"kind", h.metadata.Kind,
				"name", name,
				"namespace", namespace,
			)
			return nil, err
		}
		return nil, errorpkg.Wrapf(err, "could not find %s in store", h.metadata.Kind)
	}

	return obj, nil
}

// List retrieves all resources, optionally filtered by namespace and labels
// Implements rest.Lister
func (h *GenericRESTHandler[T]) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	// Extract label selector if provided
	var labelSelector labels.Selector
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}

	// Get namespace from context (empty string = all namespaces)
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.V(4).InfoS("Listing resources",
		"kind", h.metadata.Kind,
		"namespace", namespace,
	)

	// Query storage
	filter := v2storage.Filter{Namespace: namespace}
	items, err := h.repo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to list resource %s", h.metadata.Kind))
	}

	// Create empty list object
	listObj := h.metadata.NewListFunc()

	// Filter by label selector and resource version
	var desiredRV uint64
	if options != nil && len(options.ResourceVersion) > 0 {
		desiredRV, err = strconv.ParseUint(options.ResourceVersion, 10, 64)
		if err != nil {
			return nil, err
		}
	} else {
		desiredRV = 1
	}

	// Track highest resource version seen
	var resourceVersion uint64 = 1
	filteredItems := make([]runtime.Object, 0, len(items))

	for _, item := range items {
		// Check label selector and resource version
		allow, rv, err := h.allowObjectListWatch(item, labelSelector, desiredRV, options)
		if err != nil {
			return nil, err
		}

		if rv > resourceVersion {
			resourceVersion = rv
		}

		if allow {
			filteredItems = append(filteredItems, item)
		}
	}

	// Set items in list
	h.metadata.SetListItemsFunc(listObj, filteredItems)

	// Set list metadata
	if listMeta, ok := listObj.(metav1.ListMetaAccessor); ok {
		listMeta.GetListMeta().SetResourceVersion(strconv.FormatUint(resourceVersion, 10))
	}

	klog.V(4).InfoS("Listed resources",
		"kind", h.metadata.Kind,
		"namespace", namespace,
		"count", len(filteredItems),
	)

	return listObj, nil
}

// allowObjectListWatch determines if an object should be included in list/watch results
// based on label selector and resource version matching
func (h *GenericRESTHandler[T]) allowObjectListWatch(
	obj metav1.Object,
	labelSelector labels.Selector,
	desiredRV uint64,
	options *metainternalversion.ListOptions,
) (bool, uint64, error) {
	// Parse object's resource version
	rv, err := strconv.ParseUint(obj.GetResourceVersion(), 10, 64)
	if err != nil {
		return false, 0, err
	}

	// Check resource version match mode
	if options != nil && options.ResourceVersionMatch != "" {
		switch options.ResourceVersionMatch {
		case metav1.ResourceVersionMatchNotOlderThan:
			if rv < desiredRV {
				return false, rv, nil
			}
		case metav1.ResourceVersionMatchExact:
			if rv != desiredRV {
				return false, rv, nil
			}
		}
	}

	// Check label selector
	if labelSelector != nil && !labelSelector.Empty() {
		if !labelSelector.Matches(labels.Set(obj.GetLabels())) {
			return false, rv, nil
		}
	}

	return true, rv, nil
}

