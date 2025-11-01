package rest

import "k8s.io/apiserver/pkg/registry/rest"

type IRestHandler interface {
	rest.Storage
	rest.KindProvider
	rest.Scoper
	rest.SingularNameProvider
	rest.StandardStorage
	rest.ShortNamesProvider
}
