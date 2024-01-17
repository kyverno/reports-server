package storage

import (
	"github.com/kyverno/policy-reports/pkg/storage/api"
	"github.com/kyverno/policy-reports/pkg/storage/db"
	"github.com/kyverno/policy-reports/pkg/storage/inmemory"
	"k8s.io/klog/v2"
)

type Interface interface {
	api.Storage
}

func New(debug bool, config *db.PostgresConfig) (Interface, error) {
	klog.Infof("setting up storage, debug=%v", debug)
	if debug {
		return inmemory.New(), nil
	}
	return db.New(config)
}
