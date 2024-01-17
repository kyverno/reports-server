// Copyright 2023 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"github.com/kyverno/policy-reports/pkg/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var (
	// Scheme contains the types needed by the resource API.
	Scheme = runtime.NewScheme()
	// Codecs is a codec factory for serving the resource API.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	utilruntime.Must(v1alpha2.AddToScheme(Scheme))
	utilruntime.Must(Scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion))
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

// Build constructs APIGroupInfo the wgpolicyk8s.io API group using the given getters.
func Build(polr, cpolr rest.Storage) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v1alpha2.SchemeGroupVersion.Group, Scheme, metav1.ParameterCodec, Codecs)
	policyServerResources := map[string]rest.Storage{
		"policyreports":        polr,
		"clusterpolicyreports": cpolr,
	}
	apiGroupInfo.VersionedResourcesStorageMap[v1alpha2.SchemeGroupVersion.Version] = policyServerResources

	return apiGroupInfo
}

// Install builds the metrics for the wgpolicyk8s.io API, and then installs it into the given API policy-reports.
func Install(store storage.Interface, server *genericapiserver.GenericAPIServer) error {
	polr := PolicyReportStore(store)
	cpolr := ClusterPolicyReportStore(store)
	info := Build(polr, cpolr)
	return server.InstallAPIGroup(&info)
}
