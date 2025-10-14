package rest

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
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
	startTime := time.Now()
	isDryRun := slices.Contains(options.DryRun, "All")
	resourceType := h.metadata.Kind

	// Track in-flight requests
	h.metrics.API().IncrementInflightRequests(h.metadata.Resource, metrics.VerbCreate)
	defer h.metrics.API().DecrementInflightRequests(h.metadata.Resource, metrics.VerbCreate)

	var err error
	var statusCode string

	defer func() {
		// Record API request metrics
		h.metrics.API().RecordRequest(h.metadata.Resource, metrics.VerbCreate, statusCode, time.Since(startTime))

		klog.V(logging.LevelDebug).InfoS("Create operation completed",
			"kind", h.metadata.Kind,
			"duration", time.Since(startTime),
			"status", statusCode,
			"dryRun", isDryRun,
		)
	}()

	// Validate object
	err = createValidation(ctx, obj)
	if err != nil {
		validationErr := handleValidationErrors(err, options)
		if validationErr != nil {
			h.metrics.API().RecordValidationError(h.metadata.Resource, metrics.VerbCreate)
			statusCode = "400"
			klog.V(logging.LevelWarning).InfoS("Validation failed",
				"kind", h.metadata.Kind,
				"error", validationErr)
			return nil, validationErr
		}
	}

	// Type assert to our generic type
	resource, ok := obj.(T)
	if !ok {
		statusCode = "400"
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to validate %s", h.metadata.Kind))
	}

	resource = h.setResourceNamespaceIfNotProvided(ctx, resource)

	resource, err = h.setResourceNameIfNotProvided(resource)
	if err != nil {
		statusCode = "409"
		return nil, err
	}

	// Set creation metadata
	resource = h.setCreationMetadata(resource)

	klog.V(logging.LevelDebug).InfoS("Creating resource",
		"kind", h.metadata.Kind,
		"name", resource.GetName(),
		"namespace", resource.GetNamespace(),
		"dryRun", isDryRun,
	)

	if isDryRun {
		statusCode = "200"
		return resource, nil
	}

	// Persist if not dry-run
	opStart := time.Now()
	err = h.repo.Create(ctx, resource)
	opDuration := time.Since(opStart)

	if err != nil {
		h.metrics.Storage().RecordOperation(resourceType, metrics.OpCreate, metrics.StatusError, opDuration)
		statusCode = "500"
		if errors.IsAlreadyExists(err) {
			statusCode = "409"
		}
		klog.ErrorS(err, "Failed to create resource in storage",
			"kind", h.metadata.Kind,
			"name", resource.GetName(),
			"namespace", resource.GetNamespace())
		return nil, err
	}

	// Success
	h.metrics.Storage().RecordOperation(resourceType, metrics.OpCreate, metrics.StatusSuccess, opDuration)
	statusCode = "201"

	// Broadcast watch event
	if err := h.broadcaster.Action(watch.Added, resource); err != nil {
		klog.ErrorS(err, "Failed to broadcast watch event",
			"kind", h.metadata.Kind,
			"name", resource.GetName())
		h.metrics.Watch().RecordEventDropped(resourceType, "broadcast_error")
	} else {
		h.metrics.Watch().RecordEvent(resourceType, "added")
	}

	return resource, nil
}

// labelReports adds the ServedByReportsServer annotation
func labelReports(annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["reports.kyverno.io/served-by-reports-server"] = "true"
	return annotations
}

func handleValidationErrors(err error, options *metav1.CreateOptions) error {
	switch options.FieldValidation {
	case "Ignore":
		return nil
	case "Warn":
		// Log warning but continue
		klog.V(4).InfoS("Validation warning ignored", "error", err)
		return nil
	case "Strict":
		return err
	default:
		return err
	}
}

func (h *GenericRESTHandler[T]) setResourceNamespaceIfNotProvided(ctx context.Context, resource T) T {
	if !h.metadata.Namespaced {
		return resource
	}

	if len(resource.GetNamespace()) > 0 {
		return resource
	}

	// Get namespace from context
	namespace := genericapirequest.NamespaceValue(ctx)
	resource.SetNamespace(namespace)

	return resource
}

func (h *GenericRESTHandler[T]) setResourceNameIfNotProvided(resource T) (T, error) {

	if resource.GetName() != "" {
		return resource, nil
	}

	var nilObj T
	if resource.GetGenerateName() == "" {
		return nilObj, errors.NewConflict(
			h.metadata.ToGroupResource(),
			resource.GetName(),
			fmt.Errorf("name and generateName not provided"),
		)
	}

	resource.SetName(nameGenerator.GenerateName(resource.GetGenerateName()))

	return resource, nil
}

func (h *GenericRESTHandler[T]) setCreationMetadata(resource T) T {
	resource.SetAnnotations(labelReports(resource.GetAnnotations()))
	resource.SetGeneration(1)
	resource.SetResourceVersion(h.versioning.UseResourceVersion())
	resource.SetUID(uuid.NewUUID())
	resource.SetCreationTimestamp(metav1.Now())
	return resource
}
