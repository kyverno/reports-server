package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type cephrdb struct {
	sync.Mutex
	db *db[reportsv1.ClusterEphemeralReport]
}

func (c *cephrdb) key(name string) string {
	return fmt.Sprintf("cephr/%s", name)
}

func (c *cephrdb) List(ctx context.Context) ([]*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	klog.Infof("listing all values")

	res := make([]*reportsv1.ClusterEphemeralReport, 0, len(c.db.Keys()))
	for _, k := range c.db.Keys() {
		v, _ := c.db.Get(k)
		res = append(res, v)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("inmemory", "list", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("inmemory", "list", "ClusterEphemeralReport", time.Since(startTime))
	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cephrdb) Get(ctx context.Context, name string) (*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, _ := c.db.Get(key); val != nil {
		serverMetrics.UpdateDBRequestTotalMetrics("inmemory", "get", "ClusterEphemeralReport")
		serverMetrics.UpdateDBRequestLatencyMetrics("inmemory", "get", "ClusterEphemeralReport", time.Since(startTime))
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return nil, errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	}
}

func (c *cephrdb) Create(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	key := c.key(cephr.Name)
	klog.Infof("creating entry for key:%s", key)
	if v, _ := c.db.Get(key); v != nil {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.ClusterEphemeralReportsGR, key)
	} else {
		klog.Infof("entry created for key:%s", key)
		serverMetrics.UpdateDBRequestTotalMetrics("inmemory", "create", "ClusterEphemeralReport")
		serverMetrics.UpdateDBRequestLatencyMetrics("inmemory", "create", "ClusterEphemeralReport", time.Since(startTime))
		storageMetrics.UpdatePolicyReportMetrics("inmemory", "create", cephr, false)
		return c.db.Store(key, *cephr)
	}
}

func (c *cephrdb) Update(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()
	key := c.key(cephr.Name)
	klog.Infof("updating entry for key:%s", key)
	if v, _ := c.db.Get(key); v == nil {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	} else {
		klog.Infof("entry updated for key:%s", key)
		serverMetrics.UpdateDBRequestTotalMetrics("inmemory", "update", "ClusterEphemeralReport")
		serverMetrics.UpdateDBRequestLatencyMetrics("inmemory", "update", "ClusterEphemeralReport", time.Since(startTime))
		storageMetrics.UpdatePolicyReportMetrics("inmemory", "update", cephr, false)
		return c.db.Store(key, *cephr)
	}
}

func (c *cephrdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("deleting entry for key:%s", key)
	if v, _ := c.db.Get(key); v == nil {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	} else {
		report, err := c.Get(ctx, name)
		if err != nil {
			klog.ErrorS(err, "failed to get cephr")
			return fmt.Errorf("delete clusterephemeralreport: %v", err)
		}
		startTime := time.Now()
		c.db.Delete(key)
		klog.Infof("entry deleted for key:%s", key)
		serverMetrics.UpdateDBRequestTotalMetrics("inmemory", "delete", "ClusterEphemeralReport")
		serverMetrics.UpdateDBRequestLatencyMetrics("inmemory", "delete", "ClusterEphemeralReport", time.Since(startTime))
		storageMetrics.UpdatePolicyReportMetrics("inmemory", "delete", report, false)
		return nil
	}
}
