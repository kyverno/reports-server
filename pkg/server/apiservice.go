package server

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

type APIServices struct {
	wgpolicyApiService    apiregistrationv1.APIService
	v1ReportsApiService   apiregistrationv1.APIService
	StoreReports          bool
	StoreEphemeralReports bool
}

func BuildApiServices(serviceName string, serviceNamespace string) APIServices {
	return APIServices{
		wgpolicyApiService: apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: "v1alpha2.wgpolicyk8s.io",
				Labels: map[string]string{
					"kube-aggregator.kubernetes.io/automanaged": "false",
					"app.kubernetes.io/managed-by":              serviceName,
				},
			},
			Spec: apiregistrationv1.APIServiceSpec{
				Group:                 "wgpolicyk8s.io",
				GroupPriorityMinimum:  100,
				InsecureSkipTLSVerify: true,
				Service: &apiregistrationv1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
				Version:         "v1alpha2",
				VersionPriority: 100,
			},
		},
		v1ReportsApiService: apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: "v1.reports.kyverno.io",
				Labels: map[string]string{
					"kube-aggregator.kubernetes.io/automanaged": "false",
					"app.kubernetes.io/managed-by":              serviceName,
				},
			},
			Spec: apiregistrationv1.APIServiceSpec{
				Group:                 "reports.kyverno.io",
				GroupPriorityMinimum:  100,
				InsecureSkipTLSVerify: true,
				Service: &apiregistrationv1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
				Version:         "v1",
				VersionPriority: 100,
			},
		},
	}
}
