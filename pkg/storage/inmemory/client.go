package inmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kyverno/policy-reports/pkg/storage/api"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var (
	groupResource = v1alpha2.SchemeGroupVersion.WithResource("policyreportsa").GroupResource()
)

type inMemoryDb struct {
	sync.Mutex

	cpolrdb *cpolrdb
	polrdb  *polrdb
}

type ClusterPolicyReportsInterface interface {
	Get(ctx context.Context, name string) (v1alpha2.ClusterPolicyReport, error)
	List(ctx context.Context, name string) ([]v1alpha2.ClusterPolicyReport, error)
	Create(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Update(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Delete(ctx context.Context, name string) error
}

type cpolrdb struct {
	sync.Mutex
	db map[string]v1alpha2.ClusterPolicyReport
}

type polrdb struct {
	sync.Mutex
	db map[string]v1alpha2.PolicyReport
}

func New() api.Storage {
	inMemoryDb := &inMemoryDb{
		cpolrdb: &cpolrdb{
			db: make(map[string]v1alpha2.ClusterPolicyReport),
		},
		polrdb: &polrdb{
			db: make(map[string]v1alpha2.PolicyReport),
		},
	}
	return inMemoryDb
}

func (i *inMemoryDb) PolicyReports() api.PolicyReportsInterface {
	return i.polrdb
}

func (i *inMemoryDb) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return i.cpolrdb
}

func (i *inMemoryDb) Ready() bool {
	return true
}

func (p *polrdb) key(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (p *polrdb) List(ctx context.Context, namespace string) ([]v1alpha2.PolicyReport, error) {
	p.Lock()
	defer p.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]v1alpha2.PolicyReport, 0)

	for k, v := range p.db {
		if strings.HasPrefix(k, namespace) {
			res = append(res, v)
			klog.Infof("value found for prefix:%s, key:%s", namespace, k)
		}
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (p *polrdb) Get(ctx context.Context, name, namespace string) (v1alpha2.PolicyReport, error) {
	p.Lock()
	defer p.Unlock()

	key := p.key(name, namespace)
	klog.Infof("getting value for key:%s", key)
	if val, ok := p.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return v1alpha2.PolicyReport{}, errors.NewNotFound(groupResource, key)
	}
}

func (p *polrdb) Create(ctx context.Context, polr v1alpha2.PolicyReport) error {
	p.Lock()
	defer p.Unlock()

	key := p.key(polr.Name, polr.Namespace)
	klog.Infof("creating entry for key:%s", key)
	if _, found := p.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(groupResource, key)
	} else {
		p.db[key] = polr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (p *polrdb) Update(ctx context.Context, polr v1alpha2.PolicyReport) error {
	p.Lock()
	defer p.Unlock()

	key := p.key(polr.Name, polr.Namespace)
	klog.Infof("updating entry for key:%s", key)
	if _, found := p.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(groupResource, key)
	} else {
		p.db[key] = polr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (p *polrdb) Delete(ctx context.Context, name, namespace string) error {
	p.Lock()
	defer p.Unlock()

	key := p.key(name, namespace)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := p.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(groupResource, key)
	} else {
		delete(p.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}

func (c *cpolrdb) key(name string) string {
	return name
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
		return v1alpha2.ClusterPolicyReport{}, errors.NewNotFound(groupResource, key)
	}
}

func (c *cpolrdb) Create(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(cpolr.Name)
	klog.Infof("creating entry for key:%s", key)
	if _, found := c.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(groupResource, key)
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
		return errors.NewNotFound(groupResource, key)
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
		return errors.NewNotFound(groupResource, key)
	} else {
		delete(c.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
