package crdcoexistence

import (
	"sort"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectConflictingCRDs_NilConfig(t *testing.T) {
	if result := DetectConflictingCRDs(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestDetectFromCRDList(t *testing.T) {
	makeCRD := func(name, group string) apiextensionsv1.CustomResourceDefinition {
		return apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       apiextensionsv1.CustomResourceDefinitionSpec{Group: group},
		}
	}

	tests := []struct {
		name     string
		crds     []apiextensionsv1.CustomResourceDefinition
		expected []string
	}{
		{
			name: "no CRDs",
		},
		{
			name: "unrelated CRD",
			crds: []apiextensionsv1.CustomResourceDefinition{makeCRD("foos.example.com", "example.com")},
		},
		{
			name:     "wgpolicyk8s CRD",
			crds:     []apiextensionsv1.CustomResourceDefinition{makeCRD("policyreports.wgpolicyk8s.io", "wgpolicyk8s.io")},
			expected: []string{"/apis/wgpolicyk8s.io/"},
		},
		{
			name:     "openreports CRD",
			crds:     []apiextensionsv1.CustomResourceDefinition{makeCRD("reports.openreports.io", "openreports.io")},
			expected: []string{"/apis/openreports.io/"},
		},
		{
			name: "both groups",
			crds: []apiextensionsv1.CustomResourceDefinition{
				makeCRD("policyreports.wgpolicyk8s.io", "wgpolicyk8s.io"),
				makeCRD("reports.openreports.io", "openreports.io"),
			},
			expected: []string{"/apis/openreports.io/", "/apis/wgpolicyk8s.io/"},
		},
		{
			name: "duplicate CRDs in same group",
			crds: []apiextensionsv1.CustomResourceDefinition{
				makeCRD("policyreports.wgpolicyk8s.io", "wgpolicyk8s.io"),
				makeCRD("clusterpolicyreports.wgpolicyk8s.io", "wgpolicyk8s.io"),
			},
			expected: []string{"/apis/wgpolicyk8s.io/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFromCRDList(tt.crds)
			sort.Strings(result)
			sort.Strings(tt.expected)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}
