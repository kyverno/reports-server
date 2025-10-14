package server

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceDefinition defines everything needed to create a Kubernetes resource
// This is the ONLY thing you need to define when adding a new resource type!
type ResourceDefinition struct {
	// Basic Resource Info
	Kind         string
	SingularName string
	ShortNames   []string
	Namespaced   bool

	// API Group Info
	Group    string
	Version  string
	Resource string

	// Factory Functions
	NewFunc     func() runtime.Object
	NewListFunc func() runtime.Object

	// List Manipulation Functions
	ExtractItems func(list runtime.Object) []runtime.Object
	SetItems     func(list runtime.Object, items []runtime.Object)

	// GroupVersionKind (computed from Group, Version, Kind)
	GVK schema.GroupVersionKind

	// GroupResource (computed from Group, Resource)
	GR schema.GroupResource
}

// NewResourceDefinition creates a properly initialized resource definition
func NewResourceDefinition(
	kind, singularName string,
	shortNames []string,
	namespaced bool,
	group, version, resource string,
	newFunc, newListFunc func() runtime.Object,
	extractItems func(runtime.Object) []runtime.Object,
	setItems func(runtime.Object, []runtime.Object),
) ResourceDefinition {
	return ResourceDefinition{
		Kind:         kind,
		SingularName: singularName,
		ShortNames:   shortNames,
		Namespaced:   namespaced,
		Group:        group,
		Version:      version,
		Resource:     resource,
		NewFunc:      newFunc,
		NewListFunc:  newListFunc,
		ExtractItems: extractItems,
		SetItems:     setItems,
		GVK: schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    kind,
		},
		GR: schema.GroupResource{
			Group:    group,
			Resource: resource,
		},
	}
}
