package utils

import (
	"github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

var (
	EphemeralReportsGR         = reportsv1.Resource("ephemeralreports")
	ClusterEphemeralReportsGR  = reportsv1.Resource("clusterephemeralreports")
	PolicyReportsGR            = v1alpha2.Resource("policyreports")
	ClusterPolicyReportsGR     = v1alpha2.Resource("clusterpolicyreports")
	OpenreportsReportGR        = openreportsv1alpha1.Resource("reports")
	OpenreportsClusterReportGR = openreportsv1alpha1.Resource("clusterreports")

	EphemeralReportsGVK         = reportsv1.SchemeGroupVersion.WithKind("EphemeralReport")
	ClusterEphemeralReportsGVK  = reportsv1.SchemeGroupVersion.WithKind("ClusterEphemeralReport")
	PolicyReportsGVK            = v1alpha2.SchemeGroupVersion.WithKind("PolicyReport")
	ClusterPolicyReportsGVK     = v1alpha2.SchemeGroupVersion.WithKind("ClusterPolicyReport")
	OpenreportsReportGVK        = openreportsv1alpha1.SchemeGroupVersion.WithKind("Report")
	OpenreportsClusterReportGVK = openreportsv1alpha1.SchemeGroupVersion.WithKind("ClusterReports")
)
