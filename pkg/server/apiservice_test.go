package server

import (
	"testing"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/utils/ptr"
)

func TestBuildApiServicesSetsDefaultedPort(t *testing.T) {
	apiservices := BuildApiServices("reports-server", "reports-server", nil)

	for _, apiservice := range []apiregistrationv1.APIService{
		apiservices.wgpolicyApiService,
		apiservices.v1ReportsApiService,
		apiservices.openreportsApiService,
	} {
		if apiservice.Spec.Service == nil {
			t.Fatalf("%s: expected spec.service to be set", apiservice.Name)
		}
		if apiservice.Spec.Service.Port == nil {
			t.Fatalf("%s: expected spec.service.port to be set, got nil", apiservice.Name)
		}
		if got := *apiservice.Spec.Service.Port; got != 443 {
			t.Fatalf("%s: expected spec.service.port 443, got %d", apiservice.Name, got)
		}
	}
}

// The apiserver defaults spec.service.port to 443 on the stored object. The
// reconciler compares the desired spec against the in-cluster spec with a full
// DeepEqual, so the desired spec must already carry the defaulted value or
// every reconcile tick issues an update that changes nothing.
func TestBuildApiServicesSpecMatchesServerDefaultedSpec(t *testing.T) {
	apiservices := BuildApiServices("reports-server", "reports-server", nil)

	for _, apiservice := range []apiregistrationv1.APIService{
		apiservices.wgpolicyApiService,
		apiservices.v1ReportsApiService,
		apiservices.openreportsApiService,
	} {
		inCluster := apiservice.DeepCopy()
		inCluster.Spec.Service.Port = ptr.To[int32](443)
		if !apiequality.Semantic.DeepEqual(apiservice.Spec, inCluster.Spec) {
			t.Fatalf("%s: desired spec differs from the server-defaulted spec, reconciler would update on every tick", apiservice.Name)
		}
	}
}

func TestBuildApiServicesOwnerReferences(t *testing.T) {
	ownerRef := &metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "reports-server", UID: "uid"}
	apiservices := BuildApiServices("reports-server", "reports-server", ownerRef)
	if got := len(apiservices.wgpolicyApiService.OwnerReferences); got != 1 {
		t.Fatalf("expected 1 owner reference, got %d", got)
	}

	noOwner := BuildApiServices("reports-server", "reports-server", nil)
	if got := len(noOwner.wgpolicyApiService.OwnerReferences); got != 0 {
		t.Fatalf("expected no owner references, got %d", got)
	}
}
