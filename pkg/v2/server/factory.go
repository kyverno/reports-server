package server

import (
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/v2/rest"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"github.com/kyverno/reports-server/pkg/v2/versioning"
	restAPI "k8s.io/apiserver/pkg/registry/rest"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

// HandlerFactory creates REST handlers for all resource types
// It uses ResourceRegistry for configuration (single source of truth)
type HandlerFactory struct {
	versioning versioning.Versioning
	registry   *ResourceRegistry
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(v versioning.Versioning) *HandlerFactory {
	return &HandlerFactory{
		versioning: v,
		registry:   NewResourceRegistry(),
	}
}

// toRestMetadata converts a ResourceDefinition to rest.ResourceMetadata
func toRestMetadata(def ResourceDefinition) rest.ResourceMetadata {
	return rest.ResourceMetadata{
		Kind:             def.Kind,
		SingularName:     def.SingularName,
		ShortNames:       def.ShortNames,
		Namespaced:       def.Namespaced,
		Group:            def.Group,
		Resource:         def.Resource,
		NewFunc:          def.NewFunc,
		NewListFunc:      def.NewListFunc,
		ListItemsFunc:    def.ExtractItems,
		SetListItemsFunc: def.SetItems,
	}
}

// CreatePolicyReportHandler creates a REST handler for PolicyReport
func (f *HandlerFactory) CreatePolicyReportHandler(
	repo storage.IRepository[*v1alpha2.PolicyReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*v1alpha2.PolicyReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.PolicyReport),
	)
}

// CreateClusterPolicyReportHandler creates a REST handler for ClusterPolicyReport
func (f *HandlerFactory) CreateClusterPolicyReportHandler(
	repo storage.IRepository[*v1alpha2.ClusterPolicyReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*v1alpha2.ClusterPolicyReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.ClusterPolicyReport),
	)
}

// CreateEphemeralReportHandler creates a REST handler for EphemeralReport
func (f *HandlerFactory) CreateEphemeralReportHandler(
	repo storage.IRepository[*reportsv1.EphemeralReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*reportsv1.EphemeralReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.EphemeralReport),
	)
}

// CreateClusterEphemeralReportHandler creates a REST handler for ClusterEphemeralReport
func (f *HandlerFactory) CreateClusterEphemeralReportHandler(
	repo storage.IRepository[*reportsv1.ClusterEphemeralReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*reportsv1.ClusterEphemeralReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.ClusterEphemeralReport),
	)
}

// CreateReportHandler creates a REST handler for Report (openreports.io)
func (f *HandlerFactory) CreateReportHandler(
	repo storage.IRepository[*openreportsv1alpha1.Report],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*openreportsv1alpha1.Report](
		repo,
		f.versioning,
		toRestMetadata(f.registry.Report),
	)
}

// CreateClusterReportHandler creates a REST handler for ClusterReport (openreports.io)
func (f *HandlerFactory) CreateClusterReportHandler(
	repo storage.IRepository[*openreportsv1alpha1.ClusterReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*openreportsv1alpha1.ClusterReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.ClusterReport),
	)
}
