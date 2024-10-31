package storage

import (
	"github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/storage/db"
	"github.com/kyverno/reports-server/pkg/storage/etcd"
	"k8s.io/klog/v2"
)

type Interface interface {
	api.Versioning
	api.Storage
}

func New(embedded bool, config *db.PostgresConfig, etcdCfg *etcd.EtcdConfig) (Interface, error) {
	klog.Infof("setting up storage, embedded-db=%v, etcdconfig=%+v", embedded, etcdCfg)
	var storage api.Storage
	var err error

	if embedded {
		storage, err = etcd.New(etcdCfg)
		if err != nil {
			return nil, err
		}
	} else {
		storage, err = db.New(config)
		if err != nil {
			return nil, err
		}
	}

	return &store{
		db:         storage,
		versioning: NewVersioning(),
	}, nil
}

type store struct {
	db         api.Storage
	versioning api.Versioning
}

func (s *store) ClusterEphemeralReports() api.ClusterEphemeralReportsInterface {
	return s.db.ClusterEphemeralReports()
}

func (s *store) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return s.db.ClusterPolicyReports()
}

func (s *store) EphemeralReports() api.EphemeralReportsInterface {
	return s.db.EphemeralReports()
}

func (s *store) PolicyReports() api.PolicyReportsInterface {
	return s.db.PolicyReports()
}

func (s *store) Ready() bool {
	return s.db.Ready()
}

func (s *store) SetResourceVersion(val string) error {
	return s.versioning.SetResourceVersion(val)
}

func (s *store) UseResourceVersion() string {
	return s.versioning.UseResourceVersion()
}
