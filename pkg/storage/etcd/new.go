package etcd

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var dialTimeout = 10 * time.Second

type EtcdConfig struct {
	Endpoints string
	Insecure  bool
}

type etcdClient struct {
	polrClient           api.GenericIface[*v1alpha2.PolicyReport]
	ephrClient           api.GenericIface[*reportsv1.EphemeralReport]
	reportsClient        api.GenericIface[*openreportsv1alpha1.Report]
	cpolrClient          api.GenericClusterIface[*v1alpha2.ClusterPolicyReport]
	cephrClient          api.GenericClusterIface[*reportsv1.ClusterEphemeralReport]
	clusterReportsClient api.GenericClusterIface[*openreportsv1alpha1.ClusterReport]
}

func New(cfg *EtcdConfig) (api.Storage, error) {
	clientCfg := clientv3.Config{
		DialTimeout: dialTimeout,
		Endpoints:   strings.Split(cfg.Endpoints, ","),
	}
	if cfg.Insecure {
		clientCfg.TLS = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		}
		clientCfg.DialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	client, err := clientv3.New(clientCfg)
	if err != nil {
		return nil, err
	}

	return &etcdClient{
		polrClient:           NewObjectStoreNamespaced[*v1alpha2.PolicyReport](client, utils.PolicyReportsGVK, utils.PolicyReportsGR),
		ephrClient:           NewObjectStoreNamespaced[*reportsv1.EphemeralReport](client, utils.EphemeralReportsGVK, utils.EphemeralReportsGR),
		reportsClient:        NewObjectStoreNamespaced[*openreportsv1alpha1.Report](client, utils.OpenreportsReportGVK, utils.OpenreportsReportGR),
		cpolrClient:          NewObjectStoreCluster[*v1alpha2.ClusterPolicyReport](client, utils.ClusterPolicyReportsGVK, utils.ClusterPolicyReportsGR),
		cephrClient:          NewObjectStoreCluster[*reportsv1.ClusterEphemeralReport](client, utils.ClusterEphemeralReportsGVK, utils.ClusterEphemeralReportsGR),
		clusterReportsClient: NewObjectStoreCluster[*openreportsv1alpha1.ClusterReport](client, utils.OpenreportsClusterReportGVK, utils.OpenreportsClusterReportGR),
	}, nil
}

func (e *etcdClient) Ready(_ context.Context) bool {
	return true
}

func (e *etcdClient) PolicyReports() api.GenericIface[*v1alpha2.PolicyReport] {
	return e.polrClient
}

func (e *etcdClient) ClusterPolicyReports() api.GenericClusterIface[*v1alpha2.ClusterPolicyReport] {
	return e.cpolrClient
}

func (e *etcdClient) EphemeralReports() api.GenericIface[*reportsv1.EphemeralReport] {
	return e.ephrClient
}

func (e *etcdClient) ClusterEphemeralReports() api.GenericClusterIface[*reportsv1.ClusterEphemeralReport] {
	return e.cephrClient
}

func (e *etcdClient) Reports() api.GenericIface[*openreportsv1alpha1.Report] {
	return e.reportsClient
}

func (e *etcdClient) ClusterReports() api.GenericClusterIface[*openreportsv1alpha1.ClusterReport] {
	return e.clusterReportsClient
}
