package rest

import (
	"context"
	"fmt"
	"slices"

	v2storage "github.com/kyverno/reports-server/pkg/v2/storage"
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
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)
	filter := v2storage.NewFilter(name, namespace)

	// Get the object to delete (needed for validation and return value)
	obj, err := h.repo.Get(ctx, filter)
	if err != nil {
		klog.ErrorS(err, "Failed to find resource",
			"kind", h.metadata.Kind,
			"name", name,
			"namespace", namespace,
		)
		return nil, false, errors.NewNotFound(h.metadata.ToGroupResource(), name)
	}

	// Validate deletion
	if err := deleteValidation(ctx, obj); err != nil {
		return nil, false, err
	}

	klog.V(4).InfoS("Deleting resource",
		"kind", h.metadata.Kind,
		"name", name,
		"namespace", namespace,
	)

	if isDryRun {
		return obj, true, nil
	}

	// Delete if not dry-run
	if err := h.repo.Delete(ctx, filter); err != nil {
		if errors.IsNotFound(err) {
			return nil, false, err
		}
		return nil, false, fmt.Errorf("failed to delete %s: %w", h.metadata.Kind, err)
	}

	// Broadcast watch event
	if err := h.broadcaster.Action(watch.Deleted, obj); err != nil {
		klog.ErrorS(err, "Failed to broadcast delete event")
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
		filter := v2storage.NewFilter(itemObj.GetName(), itemObj.GetNamespace())
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
