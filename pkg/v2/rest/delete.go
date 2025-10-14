package rest

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
)

// Delete deletes a resource
// Implements rest.GracefulDeleter
func (h *GenericRESTHandler[T]) Delete(
	ctx context.Context,
	name string,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions,
) (runtime.Object, bool, error) {
	startTime := time.Now()
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)
	resourceType := h.metadata.Kind
	filter := storage.NewFilter(name, namespace)

	// Track in-flight requests
	h.metrics.API().IncrementInflightRequests(h.metadata.Resource, metrics.VerbDelete)
	defer h.metrics.API().DecrementInflightRequests(h.metadata.Resource, metrics.VerbDelete)

	var statusCode string
	defer func() {
		h.metrics.API().RecordRequest(h.metadata.Resource, metrics.VerbDelete, statusCode, time.Since(startTime))
	}()

	// Get the object to delete (needed for validation and return value)
	obj, err := h.repo.Get(ctx, filter)
	if err != nil {
		statusCode = "404"
		klog.V(logging.LevelWarning).InfoS("Resource not found for deletion",
			"kind", h.metadata.Kind,
			"name", name,
			"namespace", namespace,
		)
		return nil, false, errors.NewNotFound(h.metadata.ToGroupResource(), name)
	}

	// Validate deletion
	if err := deleteValidation(ctx, obj); err != nil {
		h.metrics.API().RecordValidationError(h.metadata.Resource, metrics.VerbDelete)
		statusCode = "422"
		klog.V(logging.LevelWarning).InfoS("Delete validation failed",
			"kind", h.metadata.Kind,
			"name", name,
			"error", err)
		return nil, false, err
	}

	klog.V(logging.LevelDebug).InfoS("Deleting resource",
		"kind", h.metadata.Kind,
		"name", name,
		"namespace", namespace,
		"dryRun", isDryRun,
	)

	if isDryRun {
		statusCode = "200"
		return obj, true, nil
	}

	// Delete if not dry-run
	opStart := time.Now()
	if err := h.repo.Delete(ctx, filter); err != nil {
		opDuration := time.Since(opStart)
		
		if errors.IsNotFound(err) {
			h.metrics.Storage().RecordOperation(resourceType, metrics.OpDelete, metrics.StatusNotFound, opDuration)
			statusCode = "404"
			return nil, false, err
		}
		
		h.metrics.Storage().RecordOperation(resourceType, metrics.OpDelete, metrics.StatusError, opDuration)
		statusCode = "500"
		klog.ErrorS(err, "Failed to delete resource from storage",
			"kind", h.metadata.Kind,
			"name", name)
		return nil, false, fmt.Errorf("failed to delete %s: %w", h.metadata.Kind, err)
	}

	h.metrics.Storage().RecordOperation(resourceType, metrics.OpDelete, metrics.StatusSuccess, time.Since(opStart))
	statusCode = "200"

	// Broadcast watch event
	if err := h.broadcaster.Action(watch.Deleted, obj); err != nil {
		klog.ErrorS(err, "Failed to broadcast delete event",
			"kind", h.metadata.Kind,
			"name", name)
		h.metrics.Watch().RecordEventDropped(resourceType, "broadcast_error")
	} else {
		h.metrics.Watch().RecordEvent(resourceType, "deleted")
	}

	return obj, true, nil
}

// DeleteCollection deletes a collection of resources
// Implements rest.CollectionDeleter
func (h *GenericRESTHandler[T]) DeleteCollection(
	ctx context.Context,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions,
	listOptions *metainternalversion.ListOptions,
) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	// List all resources matching criteria
	obj, err := h.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to list resources for deletion",
			"kind", h.metadata.Kind,
			"namespace", namespace,
		)
		return nil, errors.NewBadRequest(fmt.Sprintf("Failed to find %s", h.metadata.Kind))
	}

	items := h.metadata.ListItemsFunc(obj)
	itemObjs := h.convertRuntimeObjectsToGenericObjects(items)

	// Validate all items first (even for dry-run)
	err = h.validateDeleteForObjects(ctx, itemObjs, deleteValidation)
	if err != nil {
		return nil, err
	}

	// Return early if dry-run (after validation)
	if isDryRun {
		return obj, nil
	}

	// Delete each resource if not dry-run
	for _, itemObj := range itemObjs {
		filter := storage.NewFilter(itemObj.GetName(), itemObj.GetNamespace())
		if err := h.repo.Delete(ctx, filter); err != nil {
			klog.ErrorS(err, "Failed to delete resource",
				"kind", h.metadata.Kind,
				"name", itemObj.GetName(),
				"namespace", itemObj.GetNamespace(),
			)
			return nil, fmt.Errorf("failed to delete %s %s/%s: %w",
				h.metadata.Kind, itemObj.GetNamespace(), itemObj.GetName(), err)
		}

		klog.V(4).InfoS("Deleted resource",
			"kind", h.metadata.Kind,
			"name", itemObj.GetName(),
			"namespace", itemObj.GetNamespace(),
		)

		// Broadcast watch event
		if err := h.broadcaster.Action(watch.Deleted, itemObj); err != nil {
			klog.ErrorS(err, "Failed to broadcast delete event")
		}
	}

	return obj, nil
}

func (h *GenericRESTHandler[T]) validateDeleteForObjects(
	ctx context.Context,
	objects []T,
	deleteValidation rest.ValidateObjectFunc,
) error {
	for _, obj := range objects {
		if err := deleteValidation(ctx, obj); err != nil {
			return fmt.Errorf("validation failed for %s %s/%s: %w",
				h.metadata.Kind, obj.GetNamespace(), obj.GetName(), err)
		}
	}

	return nil
}

func (h *GenericRESTHandler[T]) convertRuntimeObjectsToGenericObjects(
	objects []runtime.Object,
) []T {
	// Pre-allocate with capacity for best case (all objects are type T)
	genericObjects := make([]T, 0, len(objects))

	for _, obj := range objects {
		typedObj, ok := obj.(T)
		if !ok {
			klog.V(4).InfoS("Skipping object with unexpected type",
				"kind", h.metadata.Kind,
				"expectedType", fmt.Sprintf("%T", *new(T)),
				"actualType", fmt.Sprintf("%T", obj),
			)
			continue
		}
		genericObjects = append(genericObjects, typedObj)
	}

	return genericObjects
}
