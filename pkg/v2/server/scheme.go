package server

import (
	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var (
	// Scheme is the runtime scheme for all API types
	Scheme = runtime.NewScheme()

	// Codecs provides serializers and deserializers for API objects
	Codecs = serializer.NewCodecFactory(Scheme)

	// ParameterCodec handles parameters in URLs
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

// init registers all API types with the scheme
func init() {
	utilruntime.Must(addKnownTypes(Scheme))
	utilruntime.Must(setPriorities(Scheme))
}

// addKnownTypes registers all API types with the scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	// wgpolicyk8s.io types
	utilruntime.Must(installWgPolicyTypesInternal(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))

	// reports.kyverno.io types
	utilruntime.Must(reportsv1.Install(scheme))

	// openreports.io types
	utilruntime.Must(openreportsv1alpha1.AddToScheme(scheme))

	// metav1 for list operations
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})

	return nil
}

// setPriorities sets version priorities for API groups
func setPriorities(scheme *runtime.Scheme) error {
	utilruntime.Must(scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion))
	utilruntime.Must(scheme.SetVersionPriority(reportsv1.SchemeGroupVersion))
	utilruntime.Must(scheme.SetVersionPriority(openreportsv1alpha1.SchemeGroupVersion))
	return nil
}

// installWgPolicyTypesInternal registers internal wgpolicyk8s.io types
func installWgPolicyTypesInternal(scheme *runtime.Scheme) error {
	schemeGroupVersion := schema.GroupVersion{
		Group:   "wgpolicyk8s.io",
		Version: runtime.APIVersionInternal,
	}

	addKnownTypes := func(s *runtime.Scheme) error {
		s.AddKnownTypes(
			schemeGroupVersion,
			&v1alpha2.ClusterPolicyReport{},
			&v1alpha2.PolicyReport{},
			&v1alpha2.ClusterPolicyReportList{},
			&v1alpha2.PolicyReportList{},
		)
		return nil
	}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	return schemeBuilder.AddToScheme(scheme)
}

// GetScheme returns the global scheme
func GetScheme() *runtime.Scheme {
	return Scheme
}

// GetCodecs returns the global codec factory
func GetCodecs() serializer.CodecFactory {
	return Codecs
}
