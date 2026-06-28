package etcd

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestClusterStoreUseResourceVersionNoPanic guards against a regression where
// NewObjectStoreCluster left the embedded *versioning.ResourceVersion nil. The
// first call to UseResourceVersion (made during Create of a cluster-scoped
// report) then dereferenced that nil pointer and panicked, taking down the
// apiserver request handler. See kyverno/reports-server#352.
func TestClusterStoreUseResourceVersionNoPanic(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "wgpolicyk8s.io", Version: "v1alpha2", Kind: "ClusterPolicyReport"}
	gr := schema.GroupResource{Group: "wgpolicyk8s.io", Resource: "clusterpolicyreports"}

	// A nil KV client is fine: UseResourceVersion only touches the in-memory
	// versioning struct and never talks to etcd.
	store := NewObjectStoreCluster[metav1.Object](nil, gvk, gr)

	if got := store.UseResourceVersion(); got == "" {
		t.Fatalf("expected a non-empty resource version, got %q", got)
	}
}
