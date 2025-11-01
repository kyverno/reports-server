package server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

// BuildAPIGroupInfo creates an APIGroupInfo for a given API group
func BuildAPIGroupInfo(
	group string,
	version string,
	resources map[string]rest.Storage,
	scheme *runtime.Scheme,
	codecs serializer.CodecFactory,
) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(
		group,
		scheme,
		metav1.ParameterCodec,
		codecs,
	)

	apiGroupInfo.VersionedResourcesStorageMap[version] = resources

	return apiGroupInfo
}
