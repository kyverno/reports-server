package api

import (
	"context"

	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type Storage interface {
	Ready() bool
	PolicyReports() PolicyReportsInterface
	ClusterPolicyReports() ClusterPolicyReportsInterface
}
type PolicyReportsInterface interface {
	Get(ctx context.Context, name, namespace string) (v1alpha2.PolicyReport, error)
	List(ctx context.Context, namespace string) ([]v1alpha2.PolicyReport, error)
	Create(ctx context.Context, polr v1alpha2.PolicyReport) error
	Update(ctx context.Context, polr v1alpha2.PolicyReport) error
	Delete(ctx context.Context, name, namespace string) error
}

type ClusterPolicyReportsInterface interface {
	Get(ctx context.Context, name string) (v1alpha2.ClusterPolicyReport, error)
	List(ctx context.Context) ([]v1alpha2.ClusterPolicyReport, error)
	Create(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Update(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Delete(ctx context.Context, name string) error
}
