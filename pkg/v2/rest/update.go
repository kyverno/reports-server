package rest

import (
	"context"
	"fmt"
	"slices"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	errorpkg "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
)

// Update updates an existing resource
// Implements rest.Updater
func (h *GenericRESTHandler[T]) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions,
) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	// Get existing object
	filter := storage.NewFilter(name, namespace)
	oldObj, err := h.repo.Get(ctx, filter)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	// Get updated object from transformer
	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	// Type assert to our generic type
	resource, ok := updatedObject.(T)
	if !ok {
		return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to validate %s", h.metadata.Kind))
	}

	// Handle force create (upsert semantics)
	if forceAllowCreate {
		resource.SetResourceVersion(h.versioning.UseResourceVersion())

		if !isDryRun {
			err = h.repo.Update(ctx, resource)
			if err != nil {
				klog.ErrorS(err, "Failed to update resource")
			}

			if err := h.broadcaster.Action(watch.Modified, resource); err != nil {
				klog.ErrorS(err, "Failed to broadcast event")
			}
		}

		return updatedObject, true, nil
	}

	// Validate update
	err = updateValidation(ctx, updatedObject, oldObj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
			// Ignore validation errors
		case "Warn":
			// Log warning but continue
			klog.V(4).InfoS("Validation warning ignored", "error", err)
		case "Strict":
			return nil, false, err
		default:
			return nil, false, err
		}
	}

	// Set resource version
	resource.SetResourceVersion(h.versioning.UseResourceVersion())

	klog.V(4).InfoS("Updating resource",
		"kind", h.metadata.Kind,
		"name", resource.GetName(),
		"namespace", resource.GetNamespace(),
	)

	// Persist if not dry-run
	if !isDryRun {
		err = h.repo.Update(ctx, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, false, err
			}
			return nil, false, errorpkg.Wrapf(err, "failed to update %s", h.metadata.Kind)
		}

		// Broadcast watch event
		if err := h.broadcaster.Action(watch.Modified, resource); err != nil {
			klog.ErrorS(err, "Failed to broadcast event")
		}
	}

	return updatedObject, false, nil
}
