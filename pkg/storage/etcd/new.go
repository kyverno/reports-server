package etcd

import (
	"errors"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var (
	etcdEndpoints = embed.DefaultListenClientURLs
	dialTimeout   = 10 * time.Second
)

type etcdClient struct {
	polrClient  ObjectStorageNamespaced[*v1alpha2.PolicyReport]
	ephrClient  ObjectStorageNamespaced[*reportsv1.EphemeralReport]
	cpolrClient ObjectStorageCluster[*v1alpha2.ClusterPolicyReport]
	cephrClient ObjectStorageCluster[*reportsv1.ClusterEphemeralReport]
}

func New() (api.Storage, error) {
	client, err := clientv3.New(clientv3.Config{
		DialTimeout: dialTimeout,
		Endpoints:   []string{etcdEndpoints},
	})
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

func StartETCDServer(stopCh <-chan struct{}, dir string) error {
	etcdConfig := embed.NewConfig()
	etcdConfig.Dir = dir
	etcd, err := embed.StartEtcd(etcdConfig)
	if err != nil {
		return err
	}
	defer etcd.Close()

	select {
	case <-etcd.Server.ReadyNotify():
		klog.Info("etcd server is running!")
	case <-time.After(100 * time.Second):
		etcd.Server.Stop()
		return errors.New("etcd server timed out and stopped!")
	}

	select {
	case <-stopCh:
		klog.Info("etcd server stopped")
		return nil
	case err := <-etcd.Err():
		klog.Error("error encountered in etcd server", err.Error())
		return err
	}
}
