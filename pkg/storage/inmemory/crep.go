package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type orcreportdb struct {
	sync.Mutex
	db map[string]*openreportsv1alpha1.ClusterReport
}

func (c *orcreportdb) key(name string) string {
	return fmt.Sprintf("cpolr/%s", name)
}

func (c *orcreportdb) List(ctx context.Context) ([]*openreportsv1alpha1.ClusterReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")

	res := make([]*openreportsv1alpha1.ClusterReport, 0, len(c.db))
	for _, val := range c.db {
		res = append(res, val)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *orcreportdb) Get(ctx context.Context, name string) (*openreportsv1alpha1.ClusterReport, error) {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, ok := c.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return nil, errors.NewNotFound(utils.ClusterPolicyReportsGR, key)
	}
}

func (c *orcreportdb) Create(ctx context.Context, cpolr *openreportsv1alpha1.ClusterReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cpolr.Name)
	klog.Infof("creating entry for key:%s", key)
	if _, found := c.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.OpenreportsClusterReportGR, key)
	} else {
		c.db[key] = cpolr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (c *orcreportdb) Update(ctx context.Context, cpolr *openreportsv1alpha1.ClusterReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cpolr.Name)
	klog.Infof("updating entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.OpenreportsClusterReportGR, key)
	} else {
		c.db[key] = cpolr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (c *orcreportdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.OpenreportsClusterReportGR, key)
	} else {
		delete(c.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
