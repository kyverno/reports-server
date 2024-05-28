package api

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type Storage interface {
	Ready() bool
	PolicyReports() PolicyReportsInterface
	ClusterPolicyReports() ClusterPolicyReportsInterface
	EphemeralReports() EphemeralReportsInterface
	ClusterEphemeralReports() ClusterEphemeralReportsInterface
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

type EphemeralReportsInterface interface {
	Get(ctx context.Context, name, namespace string) (reportsv1.EphemeralReport, error)
	List(ctx context.Context, namespace string) ([]reportsv1.EphemeralReport, error)
	Create(ctx context.Context, polr reportsv1.EphemeralReport) error
	Update(ctx context.Context, polr reportsv1.EphemeralReport) error
	Delete(ctx context.Context, name, namespace string) error
}

type ClusterEphemeralReportsInterface interface {
	Get(ctx context.Context, name string) (reportsv1.ClusterEphemeralReport, error)
	List(ctx context.Context) ([]reportsv1.ClusterEphemeralReport, error)
	Create(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error
	Update(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error
	Delete(ctx context.Context, name string) error
}
