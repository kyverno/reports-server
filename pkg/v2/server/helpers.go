package server

import (
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"k8s.io/apimachinery/pkg/runtime"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

// List manipulation helpers for all resource types
// These functions extract/set items from Kubernetes list objects

// PolicyReport helpers
func extractPolicyReportItems(list runtime.Object) []runtime.Object {
	polrList := list.(*v1alpha2.PolicyReportList)
	items := make([]runtime.Object, len(polrList.Items))
	for i := range polrList.Items {
		items[i] = &polrList.Items[i]
	}
	return items
}

func setPolicyReportItems(list runtime.Object, items []runtime.Object) {
	polrList := list.(*v1alpha2.PolicyReportList)
	polrList.Items = make([]v1alpha2.PolicyReport, len(items))
	for i, item := range items {
		polrList.Items[i] = *item.(*v1alpha2.PolicyReport)
	}
}

// ClusterPolicyReport helpers
func extractClusterPolicyReportItems(list runtime.Object) []runtime.Object {
	cpolrList := list.(*v1alpha2.ClusterPolicyReportList)
	items := make([]runtime.Object, len(cpolrList.Items))
	for i := range cpolrList.Items {
		items[i] = &cpolrList.Items[i]
	}
	return items
}

func setClusterPolicyReportItems(list runtime.Object, items []runtime.Object) {
	cpolrList := list.(*v1alpha2.ClusterPolicyReportList)
	cpolrList.Items = make([]v1alpha2.ClusterPolicyReport, len(items))
	for i, item := range items {
		cpolrList.Items[i] = *item.(*v1alpha2.ClusterPolicyReport)
	}
}

// EphemeralReport helpers
func extractEphemeralReportItems(list runtime.Object) []runtime.Object {
	ephrList := list.(*reportsv1.EphemeralReportList)
	items := make([]runtime.Object, len(ephrList.Items))
	for i := range ephrList.Items {
		items[i] = &ephrList.Items[i]
	}
	return items
}

func setEphemeralReportItems(list runtime.Object, items []runtime.Object) {
	ephrList := list.(*reportsv1.EphemeralReportList)
	ephrList.Items = make([]reportsv1.EphemeralReport, len(items))
	for i, item := range items {
		ephrList.Items[i] = *item.(*reportsv1.EphemeralReport)
	}
}

// ClusterEphemeralReport helpers
func extractClusterEphemeralReportItems(list runtime.Object) []runtime.Object {
	cephrList := list.(*reportsv1.ClusterEphemeralReportList)
	items := make([]runtime.Object, len(cephrList.Items))
	for i := range cephrList.Items {
		items[i] = &cephrList.Items[i]
	}
	return items
}

func setClusterEphemeralReportItems(list runtime.Object, items []runtime.Object) {
	cephrList := list.(*reportsv1.ClusterEphemeralReportList)
	cephrList.Items = make([]reportsv1.ClusterEphemeralReport, len(items))
	for i, item := range items {
		cephrList.Items[i] = *item.(*reportsv1.ClusterEphemeralReport)
	}
}

// Report (openreports.io) helpers
func extractReportItems(list runtime.Object) []runtime.Object {
	repList := list.(*openreportsv1alpha1.ReportList)
	items := make([]runtime.Object, len(repList.Items))
	for i := range repList.Items {
		items[i] = &repList.Items[i]
	}
	return items
}

func setReportItems(list runtime.Object, items []runtime.Object) {
	repList := list.(*openreportsv1alpha1.ReportList)
	repList.Items = make([]openreportsv1alpha1.Report, len(items))
	for i, item := range items {
		repList.Items[i] = *item.(*openreportsv1alpha1.Report)
	}
}

// ClusterReport (openreports.io) helpers
func extractClusterReportItems(list runtime.Object) []runtime.Object {
	crepList := list.(*openreportsv1alpha1.ClusterReportList)
	items := make([]runtime.Object, len(crepList.Items))
	for i := range crepList.Items {
		items[i] = &crepList.Items[i]
	}
	return items
}

func setClusterReportItems(list runtime.Object, items []runtime.Object) {
	crepList := list.(*openreportsv1alpha1.ClusterReportList)
	crepList.Items = make([]openreportsv1alpha1.ClusterReport, len(items))
	for i, item := range items {
		crepList.Items[i] = *item.(*openreportsv1alpha1.ClusterReport)
	}
}
