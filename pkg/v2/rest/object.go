package rest

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ListObject represents a Kubernetes list type (e.g., PolicyReportList)
// It must have Items field and implement runtime.Object
type ListObject interface {
	runtime.Object
	metav1.ListMetaAccessor
}

// Object represents a Kubernetes object that satisfies both metav1.Object and runtime.Object
type Object interface {
	metav1.Object
	runtime.Object
}
