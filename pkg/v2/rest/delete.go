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

	// Get the object to delete
	filter := v2storage.NewFilter(name, namespace)
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
	err = deleteValidation(ctx, obj)
	if err != nil {
		return nil, false, err
	}

	klog.V(4).InfoS("Deleting resource",
		"kind", h.metadata.Kind,
		"name", name,
		"namespace", namespace,
	)

	// Delete if not dry-run
	if !isDryRun {
		err = h.repo.Delete(ctx, filter)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, false, err
			}
			return nil, false, fmt.Errorf("failed to delete %s: %w", h.metadata.Kind, err)
		}

		// Broadcast watch event
		if err := h.broadcaster.Action(watch.Deleted, obj); err != nil {
			klog.ErrorS(err, "Failed to broadcast event")
		}
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

	// Delete each resource if not dry-run
	if !isDryRun {
		items := h.metadata.ListItemsFunc(obj)
		for _, item := range items {
			itemObj, ok := item.(T)
			if !ok {
				continue
			}

			_, isDeleted, err := h.Delete(ctx, itemObj.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete resource",
					"kind", h.metadata.Kind,
					"name", itemObj.GetName(),
					"namespace", namespace,
				)
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete %s: %s/%s",
					h.metadata.Kind, itemObj.GetNamespace(), itemObj.GetName()))
			}
		}
	}

	return obj, nil
}


