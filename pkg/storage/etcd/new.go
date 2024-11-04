package etcd

import (
	"crypto/tls"
	"strings"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var dialTimeout = 10 * time.Second

type EtcdConfig struct {
	Endpoints string
	Insecure  bool
}

type etcdClient struct {
	polrClient  ObjectStorageNamespaced[*v1alpha2.PolicyReport]
	ephrClient  ObjectStorageNamespaced[*reportsv1.EphemeralReport]
	cpolrClient ObjectStorageCluster[*v1alpha2.ClusterPolicyReport]
	cephrClient ObjectStorageCluster[*reportsv1.ClusterEphemeralReport]
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
		polrClient:  NewObjectStoreNamespaced[*v1alpha2.PolicyReport](client, utils.PolicyReportsGVK, utils.PolicyReportsGR),
		ephrClient:  NewObjectStoreNamespaced[*reportsv1.EphemeralReport](client, utils.EphemeralReportsGVK, utils.EphemeralReportsGR),
		cpolrClient: NewObjectStoreCluster[*v1alpha2.ClusterPolicyReport](client, utils.ClusterPolicyReportsGVK, utils.ClusterPolicyReportsGR),
		cephrClient: NewObjectStoreCluster[*reportsv1.ClusterEphemeralReport](client, utils.ClusterEphemeralReportsGVK, utils.ClusterEphemeralReportsGR),
	}, nil
}

func (e *etcdClient) Ready() bool {
	return true
}

func (e *etcdClient) PolicyReports() api.PolicyReportsInterface {
	return e.polrClient
}

func (e *etcdClient) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return e.cpolrClient
}

func (e *etcdClient) EphemeralReports() api.EphemeralReportsInterface {
	return e.ephrClient
}

func (e *etcdClient) ClusterEphemeralReports() api.ClusterEphemeralReportsInterface {
	return e.cephrClient
}
