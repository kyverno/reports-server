package rest

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
)

// Create creates a new resource
// Implements rest.Creater
func (h *GenericRESTHandler[T]) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions,
) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	// Validate object
	err := createValidation(ctx, obj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
			// Ignore validation errors
		case "Warn":
			// Log warning but continue
			klog.V(4).InfoS("Validation warning ignored", "error", err)
		case "Strict":
			return nil, err
		default:
			return nil, err
		}
	}

	// Type assert to our generic type
	resource, ok := obj.(T)
	if !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to validate %s", h.metadata.Kind))
	}

	// Get namespace from context
	namespace := genericapirequest.NamespaceValue(ctx)

	// Set namespace if not provided
	if h.metadata.Namespaced && len(resource.GetNamespace()) == 0 {
		resource.SetNamespace(namespace)
	}

	// Generate name if needed
	if resource.GetName() == "" {
		if resource.GetGenerateName() == "" {
			return nil, errors.NewConflict(
				h.metadata.ToGroupResource(),
				resource.GetName(),
				fmt.Errorf("name and generateName not provided"),
			)
		}
		resource.SetName(nameGenerator.GenerateName(resource.GetGenerateName()))
	}

	// Set creation metadata
	resource.SetAnnotations(labelReports(resource.GetAnnotations()))
	resource.SetGeneration(1)
	resource.SetResourceVersion(h.versioning.UseResourceVersion())
	resource.SetUID(uuid.NewUUID())
	resource.SetCreationTimestamp(metav1.Now())

	klog.V(4).InfoS("Creating resource",
		"kind", h.metadata.Kind,
		"name", resource.GetName(),
		"namespace", resource.GetNamespace(),
	)

	// Persist if not dry-run
	if !isDryRun {
		err = h.repo.Create(ctx, resource)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				return nil, err
			}
			return nil, errors.NewAlreadyExists(h.metadata.ToGroupResource(), resource.GetName())
		}

		// Broadcast watch event
		if err := h.broadcaster.Action(watch.Added, resource); err != nil {
			klog.ErrorS(err, "Failed to broadcast event")
		}
	}

	return obj, nil
}

// labelReports adds the ServedByReportsServer annotation
func labelReports(annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["reports.kyverno.io/served-by-reports-server"] = "true"
	return annotations
}

