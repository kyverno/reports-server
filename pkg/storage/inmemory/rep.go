package inmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type orreportdb struct {
	sync.Mutex
	db map[string]*openreportsv1alpha1.Report
}

func (o *orreportdb) key(name, namespace string) string {
	return fmt.Sprintf("polr/%s/%s", namespace, name)
}

func (o *orreportdb) List(ctx context.Context, namespace string) ([]*openreportsv1alpha1.Report, error) {
	o.Lock()
	defer o.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]*openreportsv1alpha1.Report, 0)

	for k, v := range o.db {
		if namespace == "" || strings.HasPrefix(strings.TrimPrefix(k, "polr/"), namespace) {
			res = append(res, v)
			klog.Infof("value found for prefix:%s, key:%s", namespace, k)
		}
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (o *orreportdb) Get(ctx context.Context, name, namespace string) (*openreportsv1alpha1.Report, error) {
	o.Lock()
	defer o.Unlock()

	key := o.key(name, namespace)
	klog.Infof("getting value for key:%s", key)
	if val, ok := o.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return nil, errors.NewNotFound(utils.OpenreportsReportGR, key)
	}
}

func (o *orreportdb) Create(ctx context.Context, polr *openreportsv1alpha1.Report) error {
	o.Lock()
	defer o.Unlock()

	key := o.key(polr.Name, polr.Namespace)
	klog.Infof("creating entry for key:%s", key)
	if _, found := o.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(utils.OpenreportsReportGR, key)
	} else {
		o.db[key] = polr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (o *orreportdb) Update(ctx context.Context, polr *openreportsv1alpha1.Report) error {
	o.Lock()
	defer o.Unlock()

	key := o.key(polr.Name, polr.Namespace)
	klog.Infof("updating entry for key:%s", key)
	if _, found := o.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.OpenreportsReportGR, key)
	} else {
		o.db[key] = polr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (o *orreportdb) Delete(ctx context.Context, name, namespace string) error {
	o.Lock()
	defer o.Unlock()

	key := o.key(name, namespace)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := o.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(utils.OpenreportsReportGR, key)
	} else {
		delete(o.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
