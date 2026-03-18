package inmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kyverno/reports-server/pkg/storage/versioning"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type genericClusterInMemStore[T any, PT interface {
	*T
	metav1.Object
}] struct {
	sync.Mutex
	*versioning.ResourceVersion
	typeName string
	gr       schema.GroupResource
	db       map[string]PT
}

func newGenericClusterInMemStore[T any, PT interface {
	*T
	metav1.Object
}](typeName string, gr schema.GroupResource) *genericClusterInMemStore[T, PT] {
	return &genericClusterInMemStore[T, PT]{
		typeName:        typeName,
		gr:              gr,
		ResourceVersion: versioning.NewVersioning(),
		db:              make(map[string]PT),
	}
}

func (c *genericClusterInMemStore[T, PT]) key(name string) string {
	return fmt.Sprintf("%s/%s", c.typeName, name)
}

func (c *genericClusterInMemStore[T, PT]) List(ctx context.Context) ([]PT, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")

	res := make([]PT, 0, len(c.db))
	for _, val := range c.db {
		res = append(res, val)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *genericClusterInMemStore[T, PT]) Get(ctx context.Context, name string) (PT, error) {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("getting value for key:%s", key)
	if val, ok := c.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	} else {
		klog.Errorf("value not found for key:%s", key)
		return nil, errors.NewNotFound(c.gr, key)
	}
}

func (c *genericClusterInMemStore[T, PT]) Create(ctx context.Context, obj PT) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(obj.GetName())
	klog.Infof("creating entry for key:%s", key)
	if _, found := c.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(c.gr, key)
	} else {
		c.db[key] = obj
		klog.Infof("entry created for key:%s", key)
		return nil
	}
}

func (c *genericClusterInMemStore[T, PT]) Update(ctx context.Context, obj PT) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(obj.GetName())
	klog.Infof("updating entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(c.gr, key)
	} else {
		c.db[key] = obj
		klog.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (c *genericClusterInMemStore[T, PT]) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	key := c.key(name)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := c.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(c.gr, key)
	} else {
		delete(c.db, key)
		klog.Infof("entry deleted for key:%s", key)
		return nil
	}
}

type genericInMemStore[T any, PT interface {
	*T
	metav1.Object
}] struct {
	sync.Mutex
	*versioning.ResourceVersion
	typeName string
	gr       schema.GroupResource
	db       map[string]PT
}

func newGenericInMemStore[T any, PT interface {
	*T
	metav1.Object
}](typeName string, gr schema.GroupResource) *genericInMemStore[T, PT] {
	return &genericInMemStore[T, PT]{
		typeName:        typeName,
		ResourceVersion: versioning.NewVersioning(),
		gr:              gr,
		db:              make(map[string]PT),
	}
}

func (g *genericInMemStore[T, PT]) key(name, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", g.typeName, namespace, name)
}

func (g *genericInMemStore[T, PT]) groupResource() schema.GroupResource {
	return schema.ParseGroupResource(g.typeName)
}

func (g *genericInMemStore[T, PT]) List(ctx context.Context, namespace string) ([]PT, error) {
	g.Lock()
	defer g.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]PT, 0)

	prefix := fmt.Sprintf("%s/", g.typeName)
	for k, v := range g.db {
		ns := strings.SplitN(strings.TrimPrefix(k, prefix), "/", 2)[0]
		if namespace == "" || ns == namespace {
			res = append(res, v)
		}
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (g *genericInMemStore[T, PT]) Get(ctx context.Context, name, namespace string) (PT, error) {
	g.Lock()
	defer g.Unlock()

	key := g.key(name, namespace)
	klog.Infof("getting value for key:%s", key)
	if val, ok := g.db[key]; ok {
		klog.Infof("value found for key:%s", key)
		return val, nil
	}
	klog.Errorf("value not found for key:%s", key)
	return nil, errors.NewNotFound(g.groupResource(), key)
}

func (g *genericInMemStore[T, PT]) Create(ctx context.Context, obj PT) error {
	g.Lock()
	defer g.Unlock()

	key := g.key(obj.GetName(), obj.GetNamespace())
	klog.Infof("creating entry for key:%s", key)
	if _, found := g.db[key]; found {
		klog.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(g.groupResource(), key)
	}
	g.db[key] = obj
	klog.Infof("entry created for key:%s", key)
	return nil
}

func (g *genericInMemStore[T, PT]) Update(ctx context.Context, obj PT) error {
	g.Lock()
	defer g.Unlock()

	key := g.key(obj.GetName(), obj.GetNamespace())
	klog.Infof("updating entry for key:%s", key)
	if _, found := g.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(g.groupResource(), key)
	}
	g.db[key] = obj
	klog.Infof("entry updated for key:%s", key)
	return nil
}

func (g *genericInMemStore[T, PT]) Delete(ctx context.Context, name, namespace string) error {
	g.Lock()
	defer g.Unlock()

	key := g.key(name, namespace)
	klog.Infof("deleting entry for key:%s", key)
	if _, found := g.db[key]; !found {
		klog.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(g.groupResource(), key)
	}
	delete(g.db, key)
	klog.Infof("entry deleted for key:%s", key)
	return nil
}
