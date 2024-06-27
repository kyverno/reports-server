package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type cpolrdb struct {
	sync.Mutex
	db map[string]v1alpha2.ClusterPolicyReport
}

func (c *cpolrdb) key(name string) string {
	return fmt.Sprintf("cpolr/%s", name)
}

func (c *cpolrdb) List(ctx context.Context) ([]v1alpha2.ClusterPolicyReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")

	res := make([]v1alpha2.ClusterPolicyReport, 0, len(c.db))
	for _, val := range c.db {
		res = append(res, val)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cpolrdb) Get(ctx context.Context, name string) (v1alpha2.ClusterPolicyReport, error) {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, ok := c.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return v1alpha2.ClusterPolicyReport{}, errors.NewNotFound(utils.ClusterPolicyReportsGR, key)
	}
}

func (c *cpolrdb) Create(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cpolr.Name)
	klog.Infof("creating entry for key:%s", key)
	if _, found := c.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.ClusterPolicyReportsGR, key)
	} else {
		c.db[key] = cpolr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (c *cpolrdb) Update(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cpolr.Name)
	klog.Infof("updating entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterPolicyReportsGR, key)
	} else {
		c.db[key] = cpolr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (c *cpolrdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterPolicyReportsGR, key)
	} else {
		delete(c.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
