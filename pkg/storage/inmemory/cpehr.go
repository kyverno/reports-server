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
	db map[string]*reportsv1.ClusterEphemeralReport
}

func (c *cephrdb) key(name string) string {
	return fmt.Sprintf("cephr/%s", name)
}

func (c *cephrdb) List(ctx context.Context) ([]*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")

	res := make([]*reportsv1.ClusterEphemeralReport, 0, len(c.db))
	for _, val := range c.db {
		res = append(res, val)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cephrdb) Get(ctx context.Context, name string) (*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, ok := c.db[key]; ok {
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

	key := c.key(cephr.Name)
	klog.Infof("creating entry for key:%s", key)
	if _, found := c.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.ClusterEphemeralReportsGR, key)
	} else {
		c.db[key] = cephr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (c *cephrdb) Update(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cephr.Name)
	klog.Infof("updating entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	} else {
		c.db[key] = cephr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (c *cephrdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.ClusterEphemeralReportsGR, key)
	} else {
		delete(c.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
