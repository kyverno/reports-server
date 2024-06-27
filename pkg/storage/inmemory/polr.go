package inmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type polrdb struct {
	sync.Mutex
	db map[string]v1alpha2.PolicyReport
}

func (p *polrdb) key(name, namespace string) string {
	return fmt.Sprintf("polr/%s/%s", namespace, name)
}

func (p *polrdb) List(ctx context.Context, namespace string) ([]v1alpha2.PolicyReport, error) {
	p.Lock()
	defer p.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]v1alpha2.PolicyReport, 0)

	for k, v := range p.db {
		if namespace == "" || strings.HasPrefix(k, namespace) {
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
		return v1alpha2.PolicyReport{}, errors.NewNotFound(utils.PolicyReportsGR, key)
	}
}

func (p *polrdb) Create(ctx context.Context, polr v1alpha2.PolicyReport) error {
	p.Lock()
	defer p.Unlock()

	key := p.key(polr.Name, polr.Namespace)
	klog.Infof("creating entry for key:%s", key)
	if _, found := p.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.PolicyReportsGR, key)
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
		return errors.NewNotFound(utils.PolicyReportsGR, key)
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
		return errors.NewNotFound(utils.PolicyReportsGR, key)
	} else {
		delete(p.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
