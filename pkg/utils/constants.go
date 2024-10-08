package utils

import (
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"k8s.io/api/resource/v1alpha2"
)

var (
	EphemeralReportsGR        = reportsv1.Resource("ephemeralreports")
	ClusterEphemeralReportsGR = reportsv1.Resource("clusterephemeralreports")
	PolicyReportsGR           = v1alpha2.Resource("policyreports")
	ClusterPolicyReportsGR    = v1alpha2.Resource("clusterephemeralreports")

	EphemeralReportsGVK        = reportsv1.SchemeGroupVersion.WithKind("EphemeralReport")
	ClusterEphemeralReportsGVK = reportsv1.SchemeGroupVersion.WithKind("ClusterEphemeralReport")
	PolicyReportsGVK           = v1alpha2.SchemeGroupVersion.WithKind("PolicyReport")
	ClusterPolicyReportsGVK    = v1alpha2.SchemeGroupVersion.WithKind("ClusterEphemeralReport")
)
