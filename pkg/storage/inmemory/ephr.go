package inmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type ephrdb struct {
	sync.Mutex
	db map[string]reportsv1.EphemeralReport
}

func (e *ephrdb) key(name, namespace string) string {
	return fmt.Sprintf("ephr/%s/%s", namespace, name)
}

func (e *ephrdb) List(ctx context.Context, namespace string) ([]reportsv1.EphemeralReport, error) {
	e.Lock()
	defer e.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]reportsv1.EphemeralReport, 0)

	for k, v := range e.db {
		if strings.HasPrefix(k, namespace) {
			res = append(res, v)
			klog.Infof("value found for prefix:%s, key:%s", namespace, k)
		}
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (e *ephrdb) Get(ctx context.Context, name, namespace string) (reportsv1.EphemeralReport, error) {
	e.Lock()
	defer e.Unlock()

	key := e.key(name, namespace)
	klog.Infof("getting value for key:%s", key)
	if val, ok := e.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return reportsv1.EphemeralReport{}, errors.NewNotFound(groupResource, key)
	}
}

func (e *ephrdb) Create(ctx context.Context, ephr reportsv1.EphemeralReport) error {
	e.Lock()
	defer e.Unlock()

	key := e.key(ephr.Name, ephr.Namespace)
	klog.Infof("creating entry for key:%s", key)
	if _, found := e.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(groupResource, key)
	} else {
		e.db[key] = ephr
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (e *ephrdb) Update(ctx context.Context, ephr reportsv1.EphemeralReport) error {
	e.Lock()
	defer e.Unlock()

	key := e.key(ephr.Name, ephr.Namespace)
	klog.Infof("updating entry for key:%s", key)
	if _, found := e.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(groupResource, key)
	} else {
		e.db[key] = ephr
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (e *ephrdb) Delete(ctx context.Context, name, namespace string) error {
	e.Lock()
	defer e.Unlock()

	key := e.key(name, namespace)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := e.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(groupResource, key)
	} else {
		delete(e.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}
