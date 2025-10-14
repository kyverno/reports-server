package rest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	"github.com/kyverno/reports-server/pkg/v2/storage"
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
	startTime := time.Now()
	namespace := genericapirequest.NamespaceValue(ctx)
	resourceType := h.metadata.Kind

	// Track in-flight requests
	h.metrics.API().IncrementInflightRequests(h.metadata.Resource, metrics.VerbGet)
	defer h.metrics.API().DecrementInflightRequests(h.metadata.Resource, metrics.VerbGet)

	var statusCode string
	defer func() {
		h.metrics.API().RecordRequest(h.metadata.Resource, metrics.VerbGet, statusCode, time.Since(startTime))
	}()

	klog.V(logging.LevelDebug).InfoS("Getting resource",
		"kind", h.metadata.Kind,
		"name", name,
		"namespace", namespace,
	)

	filter := storage.NewFilter(name, namespace)
	
	// Measure storage operation
	opStart := time.Now()
	obj, err := h.repo.Get(ctx, filter)
	opDuration := time.Since(opStart)
	
	if err != nil {
		if errors.IsNotFound(err) {
			h.metrics.Storage().RecordOperation(resourceType, metrics.OpGet, metrics.StatusNotFound, opDuration)
			statusCode = "404"
			klog.V(logging.LevelDebug).InfoS("Resource not found",
				"kind", h.metadata.Kind,
				"name", name,
				"namespace", namespace,
			)
			return nil, err
		}
		h.metrics.Storage().RecordOperation(resourceType, metrics.OpGet, metrics.StatusError, opDuration)
		statusCode = "500"
		klog.ErrorS(err, "Failed to get resource from storage",
			"kind", h.metadata.Kind,
			"name", name,
			"namespace", namespace)
		return nil, errorpkg.Wrapf(err, "could not find %s in store", h.metadata.Kind)
	}

	// Success
	h.metrics.Storage().RecordOperation(resourceType, metrics.OpGet, metrics.StatusSuccess, opDuration)
	statusCode = "200"

	return obj, nil
}

// List retrieves all resources, optionally filtered by namespace and labels
// Implements rest.Lister
func (h *GenericRESTHandler[T]) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	startTime := time.Now()
	namespace := genericapirequest.NamespaceValue(ctx)
	resourceType := h.metadata.Kind

	// Track in-flight requests
	h.metrics.API().IncrementInflightRequests(h.metadata.Resource, metrics.VerbList)
	defer h.metrics.API().DecrementInflightRequests(h.metadata.Resource, metrics.VerbList)

	var statusCode string
	defer func() {
		h.metrics.API().RecordRequest(h.metadata.Resource, metrics.VerbList, statusCode, time.Since(startTime))
	}()

	klog.V(logging.LevelDebug).InfoS("Listing resources",
		"kind", h.metadata.Kind,
		"namespace", namespace,
	)

	// Step 1: Fetch all items from storage
	filter := storage.Filter{Namespace: namespace}
	
	opStart := time.Now()
	allItems, err := h.repo.List(ctx, filter)
	opDuration := time.Since(opStart)
	
	if err != nil {
		h.metrics.Storage().RecordOperation(resourceType, metrics.OpList, metrics.StatusError, opDuration)
		statusCode = "500"
		klog.ErrorS(err, "Failed to list resources from storage",
			"kind", h.metadata.Kind,
			"namespace", namespace)
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to list resource %s", h.metadata.Kind))
	}

	h.metrics.Storage().RecordOperation(resourceType, metrics.OpList, metrics.StatusSuccess, opDuration)

	// Step 2: Filter items based on user criteria (labels, resource version)
	matchingItems, collectionVersion := h.filterItems(allItems, options)

	// Record filter efficiency
	h.metrics.Storage().ObserveFilterEfficiency(resourceType, len(matchingItems), len(allItems))

	// Step 3: Build Kubernetes list response
	listResponse := h.buildListObject(matchingItems, collectionVersion)

	statusCode = "200"

	klog.V(logging.LevelDebug).InfoS("Listed resources",
		"kind", h.metadata.Kind,
		"namespace", namespace,
		"total", len(allItems),
		"matching", len(matchingItems),
		"efficiency", fmt.Sprintf("%.2f%%", float64(len(matchingItems))*100/float64(max(len(allItems), 1))),
	)

	// Update inventory metrics
	if namespace == "" {
		// Cluster-wide list - update total count
		h.metrics.Storage().SetReportsTotal(resourceType, h.metadata.Resource, len(allItems))
	}

	return listResponse, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// filterItems filters items by label selector and resource version
// Returns: matching items + the highest resource version seen (for list metadata)
//
// WHY track highest version?
// Kubernetes list responses include a "resourceVersion" that represents the
// state of the collection. We track the HIGHEST version from ALL items
// (even filtered ones) because that's the "version" of the data store at query time.
func (h *GenericRESTHandler[T]) filterItems(
	allItems []T,
	options *metainternalversion.ListOptions,
) ([]runtime.Object, uint64) {
	var matchingItems []runtime.Object
	var highestVersion uint64 = 1

	// Extract filter criteria
	labelSelector := h.getLabelSelector(options)
	desiredVersion := h.getDesiredVersion(options)
	versionMatchMode := h.getVersionMatchMode(options)

	for _, item := range allItems {
		// Track highest version from ALL items (for list metadata)
		itemVersion := h.getItemVersion(item)
		if itemVersion > highestVersion {
			highestVersion = itemVersion
		}

		// Filter 1: Does item match label selector?
		if !h.itemMatchesLabels(item, labelSelector) {
			continue // Skip this item
		}

		// Filter 2: Does item match resource version criteria?
		if !h.itemMatchesVersion(item, desiredVersion, versionMatchMode) {
			continue // Skip this item
		}

		// Item passed all filters - include it
		matchingItems = append(matchingItems, item)
	}

	return matchingItems, highestVersion
}

// buildListObject creates the Kubernetes list response
func (h *GenericRESTHandler[T]) buildListObject(items []runtime.Object, resourceVersion uint64) runtime.Object {
	listObj := h.metadata.NewListFunc()

	// Set items
	h.metadata.SetListItemsFunc(listObj, items)

	// Set metadata (including resource version)
	if listMeta, ok := listObj.(metav1.ListMetaAccessor); ok {
		listMeta.GetListMeta().SetResourceVersion(strconv.FormatUint(resourceVersion, 10))
	}

	return listObj
}

// getLabelSelector extracts label selector from options
func (h *GenericRESTHandler[T]) getLabelSelector(options *metainternalversion.ListOptions) labels.Selector {
	if options == nil || options.LabelSelector == nil {
		return nil
	}
	return options.LabelSelector
}

// getDesiredVersion extracts desired resource version from options
func (h *GenericRESTHandler[T]) getDesiredVersion(options *metainternalversion.ListOptions) uint64 {
	if options == nil || options.ResourceVersion == "" {
		return 0 // No version specified = accept all
	}

	version, err := strconv.ParseUint(options.ResourceVersion, 10, 64)
	if err != nil {
		klog.V(4).InfoS("Invalid resource version in options, ignoring",
			"resourceVersion", options.ResourceVersion,
			"error", err)
		return 0
	}

	return version
}

// getVersionMatchMode extracts version match mode from options
func (h *GenericRESTHandler[T]) getVersionMatchMode(options *metainternalversion.ListOptions) metav1.ResourceVersionMatch {
	if options == nil {
		return ""
	}
	return options.ResourceVersionMatch
}

// getItemVersion gets the resource version from an item
func (h *GenericRESTHandler[T]) getItemVersion(item T) uint64 {
	version, err := strconv.ParseUint(item.GetResourceVersion(), 10, 64)
	if err != nil {
		// Log but don't fail - treat as version 0
		klog.V(4).InfoS("Failed to parse item resource version",
			"name", item.GetName(),
			"namespace", item.GetNamespace(),
			"error", err)
		return 0
	}
	return version
}

// itemMatchesLabels checks if item matches the label selector
func (h *GenericRESTHandler[T]) itemMatchesLabels(item T, selector labels.Selector) bool {
	// No selector = match all
	if selector == nil || selector.Empty() {
		return true
	}

	// Check if item's labels match the selector
	return selector.Matches(labels.Set(item.GetLabels()))
}

// itemMatchesVersion checks if item matches the resource version criteria
func (h *GenericRESTHandler[T]) itemMatchesVersion(item T, desiredVersion uint64, matchMode metav1.ResourceVersionMatch) bool {
	// No version specified = match all
	if desiredVersion == 0 {
		return true
	}

	itemVersion := h.getItemVersion(item)

	switch matchMode {
	case metav1.ResourceVersionMatchNotOlderThan:
		return itemVersion >= desiredVersion
	case metav1.ResourceVersionMatchExact:
		return itemVersion == desiredVersion
	default:
		return true // No match mode = match all
	}
}
