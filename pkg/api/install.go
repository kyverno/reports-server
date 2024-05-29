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
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage"
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
	utilruntime.Must(installWgPolicyTypesInternal(Scheme))
	utilruntime.Must(v1alpha2.AddToScheme(Scheme))
	utilruntime.Must(Scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion))
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	utilruntime.Must(reportsv1.Install(Scheme))
	utilruntime.Must(Scheme.SetVersionPriority(reportsv1.SchemeGroupVersion))
}

// BuildPolicyReports constructs APIGroupInfo the wgpolicyk8s.io API group using the given getters.
func BuildPolicyReports(polr, cpolr rest.Storage) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v1alpha2.SchemeGroupVersion.Group, Scheme, metav1.ParameterCodec, Codecs)
	policyReportsResources := map[string]rest.Storage{
		"policyreports":        polr,
		"clusterpolicyreports": cpolr,
	}
	apiGroupInfo.VersionedResourcesStorageMap[v1alpha2.SchemeGroupVersion.Version] = policyReportsResources
	apiGroupInfo.NegotiatedSerializer = DefaultSubsetNegotiatedSerializer(Codecs)

	return apiGroupInfo
}

// BuildEphemeralReports constructs APIGroupInfo the reports.kyverno.io API group using the given getters.
func BuildEphemeralReports(ephr, cephr rest.Storage) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(reportsv1.SchemeGroupVersion.Group, Scheme, metav1.ParameterCodec, Codecs)
	ephemeralReportsResources := map[string]rest.Storage{
		"ephemeralreports":        ephr,
		"clusterephemeralreports": cephr,
	}
	apiGroupInfo.VersionedResourcesStorageMap[reportsv1.SchemeGroupVersion.Version] = ephemeralReportsResources
	apiGroupInfo.NegotiatedSerializer = DefaultSubsetNegotiatedSerializer(Codecs)

	return apiGroupInfo
}

// Install builds the reports for the wgpolicyk8s.io and reports.kyverno.io API, and then installs it into the given API reports-server.
func Install(store storage.Interface, server *genericapiserver.GenericAPIServer) error {
	polr := PolicyReportStore(store)
	cpolr := ClusterPolicyReportStore(store)

	polrInfo := BuildPolicyReports(polr, cpolr)
	err := server.InstallAPIGroup(&polrInfo)
	if err != nil {
		return err
	}

	ephr := EphemeralReportStore(store)
	cephr := ClusterEphemeralReportStore(store)

	ephrInfo := BuildEphemeralReports(ephr, cephr)
	err = server.InstallAPIGroup(&ephrInfo)
	if err != nil {
		return err
	}

	return nil
}

func installWgPolicyTypesInternal(s *runtime.Scheme) error {
	var SchemeGroupVersion = schema.GroupVersion{Group: "wgpolicyk8s.io", Version: runtime.APIVersionInternal}
	addKnownTypes := func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(SchemeGroupVersion,
			&v1alpha2.ClusterPolicyReport{},
			&v1alpha2.PolicyReport{},
			&v1alpha2.ClusterPolicyReportList{},
			&v1alpha2.PolicyReportList{},
		)
		return nil
	}
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	utilruntime.Must(SchemeBuilder.AddToScheme(s))
	return nil
}
