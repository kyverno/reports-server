package api

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type Storage interface {
	Ready(context.Context) bool
	PolicyReports() GenericIface[*v1alpha2.PolicyReport]
	ClusterPolicyReports() GenericClusterIface[*v1alpha2.ClusterPolicyReport]
	EphemeralReports() GenericIface[*reportsv1.EphemeralReport]
	ClusterEphemeralReports() GenericClusterIface[*reportsv1.ClusterEphemeralReport]
	Reports() GenericIface[*openreportsv1alpha1.Report]
	ClusterReports() GenericClusterIface[*openreportsv1alpha1.ClusterReport]
}

type GenericIface[T metav1.Object] interface {
	Get(ctx context.Context, name, ns string) (T, error)
	List(ctx context.Context, ns string) ([]T, error)
	Create(ctx context.Context, obj T) error
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name, ns string) error
	UseResourceVersion() string
	SetResourceVersion(string) error
}

type GenericClusterIface[T metav1.Object] interface {
	Get(ctx context.Context, name string) (T, error)
	List(ctx context.Context) ([]T, error)
	Create(ctx context.Context, obj T) error
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name string) error
	UseResourceVersion() string
	SetResourceVersion(string) error
}
