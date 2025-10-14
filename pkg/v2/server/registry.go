package server

import (
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"k8s.io/apimachinery/pkg/runtime"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

// ResourceRegistry holds all resource definitions
// This is the SINGLE SOURCE OF TRUTH for all resource types
type ResourceRegistry struct {
	PolicyReport           ResourceDefinition
	ClusterPolicyReport    ResourceDefinition
	EphemeralReport        ResourceDefinition
	ClusterEphemeralReport ResourceDefinition
	Report                 ResourceDefinition
	ClusterReport          ResourceDefinition
}

// NewResourceRegistry creates the registry with all resource definitions
// TO ADD A NEW RESOURCE: Just add one entry here!
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		PolicyReport: NewResourceDefinition(
			"PolicyReport",
			"policyreport",
			[]string{"polr"},
			true, // namespaced
			GroupWGPolicy,
			"v1alpha2",
			"policyreports",
			func() runtime.Object { return &v1alpha2.PolicyReport{} },
			func() runtime.Object { return &v1alpha2.PolicyReportList{} },
			extractPolicyReportItems,
			setPolicyReportItems,
		),

		ClusterPolicyReport: NewResourceDefinition(
			"ClusterPolicyReport",
			"clusterpolicyreport",
			[]string{"cpolr"},
			false, // cluster-scoped
			GroupWGPolicy,
			"v1alpha2",
			"clusterpolicyreports",
			func() runtime.Object { return &v1alpha2.ClusterPolicyReport{} },
			func() runtime.Object { return &v1alpha2.ClusterPolicyReportList{} },
			extractClusterPolicyReportItems,
			setClusterPolicyReportItems,
		),

		EphemeralReport: NewResourceDefinition(
			"EphemeralReport",
			"ephemeralreport",
			[]string{"ephr"},
			true,
			GroupKyvernoReports,
			"v1",
			"ephemeralreports",
			func() runtime.Object { return &reportsv1.EphemeralReport{} },
			func() runtime.Object { return &reportsv1.EphemeralReportList{} },
			extractEphemeralReportItems,
			setEphemeralReportItems,
		),

		ClusterEphemeralReport: NewResourceDefinition(
			"ClusterEphemeralReport",
			"clusterephemeralreport",
			[]string{"cephr"},
			false,
			GroupKyvernoReports,
			"v1",
			"clusterephemeralreports",
			func() runtime.Object { return &reportsv1.ClusterEphemeralReport{} },
			func() runtime.Object { return &reportsv1.ClusterEphemeralReportList{} },
			extractClusterEphemeralReportItems,
			setClusterEphemeralReportItems,
		),

		Report: NewResourceDefinition(
			"Report",
			"report",
			[]string{"rep"},
			true,
			GroupOpenReports,
			"v1alpha1",
			"reports",
			func() runtime.Object { return &openreportsv1alpha1.Report{} },
			func() runtime.Object { return &openreportsv1alpha1.ReportList{} },
			extractReportItems,
			setReportItems,
		),

		ClusterReport: NewResourceDefinition(
			"ClusterReport",
			"clusterreport",
			[]string{"crep"},
			false,
			GroupOpenReports,
			"v1alpha1",
			"clusterreports",
			func() runtime.Object { return &openreportsv1alpha1.ClusterReport{} },
			func() runtime.Object { return &openreportsv1alpha1.ClusterReportList{} },
			extractClusterReportItems,
			setClusterReportItems,
		),
	}
}

// GetAllDefinitions returns all resource definitions as a slice
func (r *ResourceRegistry) GetAllDefinitions() []ResourceDefinition {
	return []ResourceDefinition{
		r.PolicyReport,
		r.ClusterPolicyReport,
		r.EphemeralReport,
		r.ClusterEphemeralReport,
		r.Report,
		r.ClusterReport,
	}
}

// GetByAPIGroup returns definitions grouped by API group
func (r *ResourceRegistry) GetByAPIGroup() map[string][]ResourceDefinition {
	return map[string][]ResourceDefinition{
		GroupWGPolicy: {
			r.PolicyReport,
			r.ClusterPolicyReport,
		},
		GroupKyvernoReports: {
			r.EphemeralReport,
			r.ClusterEphemeralReport,
		},
		GroupOpenReports: {
			r.Report,
			r.ClusterReport,
		},
	}
}
