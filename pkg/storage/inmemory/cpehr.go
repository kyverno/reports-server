package inmemory

import (
	"context"
	"fmt"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
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

func (c *cephrdb) List(ctx context.Context) ([]reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")

	res := make([]reportsv1.ClusterEphemeralReport, 0, len(c.db.Keys()))
	for _, k := range c.db.Keys() {
		v, _ := c.db.Get(k)
		res = append(res, *v)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cephrdb) Get(ctx context.Context, name string) (reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, _ := c.db.Get(key); val != nil {
		klog.Infof("value found for key:%s", key)
		return *val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return reportsv1.ClusterEphemeralReport{}, errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	}
}

func (c *cephrdb) Create(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cephr.Name)
	klog.Infof("creating entry for key:%s", key)
	if v, _ := c.db.Get(key); v != nil {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.ClusterEphemeralReportsGR, key)
	} else {
		klog.Infof("entry created for key:%s", key)
		return c.db.Store(key, cephr)
	}
}

func (c *cephrdb) Update(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cephr.Name)
	klog.Infof("updating entry for key:%s", key)
	if v, _ := c.db.Get(key); v == nil {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	} else {
		klog.Infof("entry updated for key:%s", key)
		return c.db.Store(key, cephr)
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
		c.db.Delete(key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
