package storage

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/storage/db"
	"github.com/kyverno/reports-server/pkg/storage/etcd"
	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

func New(embedded bool, config *db.PostgresConfig, etcdCfg *etcd.EtcdConfig, clusterUID string, clusterName string) (api.Storage, error) {
	klog.Infof("setting up storage, embedded-db=%v, etcdconfig=%+v", embedded, etcdCfg)
	var storage api.Storage
	var err error

	if embedded {
		storage, err = etcd.New(etcdCfg)
		if err != nil {
			return nil, err
		}
	} else {
		storage, err = db.New(config, clusterUID, clusterName)
		if err != nil {
			return nil, err
		}
	}

	return &store{
		db: storage,
	}, nil
}

type store struct {
	db api.Storage
}

func (s *store) PolicyReports() api.GenericIface[*v1alpha2.PolicyReport] {
	return s.db.PolicyReports()
}

func (s *store) ClusterPolicyReports() api.GenericClusterIface[*v1alpha2.ClusterPolicyReport] {
	return s.db.ClusterPolicyReports()
}

func (s *store) EphemeralReports() api.GenericIface[*reportsv1.EphemeralReport] {
	return s.db.EphemeralReports()
}

func (s *store) ClusterEphemeralReports() api.GenericClusterIface[*reportsv1.ClusterEphemeralReport] {
	return s.db.ClusterEphemeralReports()
}

func (s *store) Reports() api.GenericIface[*openreportsv1alpha1.Report] {
	return s.db.Reports()
}

func (s *store) ClusterReports() api.GenericClusterIface[*openreportsv1alpha1.ClusterReport] {
	return s.db.ClusterReports()
}

func (s *store) Ready(ctx context.Context) bool {
	return s.db.Ready(ctx)
}
