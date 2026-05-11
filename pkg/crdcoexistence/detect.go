package crdcoexistence

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var conflictGroups = map[string]bool{
	"wgpolicyk8s.io": true,
	"openreports.io": true,
}

// DetectConflictingCRDs checks the cluster for CRDs in API groups that
// reports-server also serves via APIService. When both a CRD and an APIService
// exist for the same group/version, the kube-apiserver's OpenAPI aggregator
// produces duplicate-path errors (K8s 1.28+). For each conflicting group found,
// the corresponding OpenAPI path prefix is returned so it can be added to
// IgnorePrefixes — suppressing reports-server's OpenAPI paths for that group
// while keeping the APIService request routing fully functional.
func DetectConflictingCRDs(restConfig *rest.Config) []string {
	if restConfig == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		klog.V(2).InfoS("unable to create apiextensions client for CRD conflict detection, skipping", "error", err)
		return nil
	}

	crdList, err := client.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(2).InfoS("unable to list CRDs for conflict detection, skipping", "error", err)
		return nil
	}

	return detectFromCRDList(crdList.Items)
}

func detectFromCRDList(crds []apiextensionsv1.CustomResourceDefinition) []string {
	seen := make(map[string]bool)
	var prefixes []string

	for _, crd := range crds {
		group := crd.Spec.Group
		if !conflictGroups[group] || seen[group] {
			continue
		}
		seen[group] = true
		klog.InfoS("detected third-party CRD for API group served by reports-server, suppressing OpenAPI paths to avoid duplicate-path errors",
			"group", group)
		prefixes = append(prefixes, "/apis/"+group+"/")
	}
	return prefixes
}
