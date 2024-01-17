/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package install installs the experimental API group, making it available as
// an option to all of the API encoding/decoding machinery.
package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var (
	// Scheme contains the types needed by the resource API.
	Scheme = runtime.NewScheme()
	// Codecs is a codec factory for serving the resource API.
	Codecs = serializer.NewCodecFactory(Scheme)
	// SchemeBuilder is the scheme builder with scheme init functions to run for this API package
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is a common registration function for mapping packaged scoped group & version keys to a scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(v1alpha2.SchemeGroupVersion,
		&v1alpha2.ClusterPolicyReport{},
		&v1alpha2.ClusterPolicyReportList{},
		&v1alpha2.PolicyReport{},
		&v1alpha2.PolicyReportList{},
	)
	return nil
}

func Install(scheme *runtime.Scheme) {
	utilruntime.Must(AddToScheme(Scheme))
	utilruntime.Must(Scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion))
}
