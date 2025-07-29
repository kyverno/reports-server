package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type ObjectStorageNamespaced[T metav1.Object] interface {
	Get(ctx context.Context, name, namespace string) (T, error)
	List(ctx context.Context, namespace string) ([]T, error)
	Create(ctx context.Context, obj T) error
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name, namespace string) error
}

type objectStoreNamespaced[T metav1.Object] struct {
	sync.Mutex
	namespaced bool
	etcdclient clientv3.KV
	gvk        schema.GroupVersionKind
	gr         schema.GroupResource
}

func NewObjectStoreNamespaced[T metav1.Object](client clientv3.KV, gvk schema.GroupVersionKind, gr schema.GroupResource) ObjectStorageNamespaced[T] {
	return &objectStoreNamespaced[T]{
		namespaced: true,
		etcdclient: client,
		gvk:        gvk,
		gr:         gr,
	}
}

func (o *objectStoreNamespaced[T]) getPrefix(namespace string) string {
	if len(namespace) != 0 {
		return fmt.Sprintf("%s/%s/%s/%s/", o.gvk.Group, o.gvk.Version, o.gvk.Kind, namespace)
	}
	return fmt.Sprintf("%s/%s/%s/", o.gvk.Group, o.gvk.Version, o.gvk.Kind)
}

func (o *objectStoreNamespaced[T]) getKey(name, namespace string) string {
	return fmt.Sprintf("%s%s", o.getPrefix(namespace), name)
}

func (o *objectStoreNamespaced[T]) Get(ctx context.Context, name, namespace string) (T, error) {
	o.Lock()
	defer o.Unlock()
	startTime := time.Now()
	var obj T
	key := o.getKey(name, namespace)
	resp, err := o.etcdclient.Get(ctx, key)
	if err != nil {
		klog.ErrorS(err, "failed to get report kind=%s", o.gvk.String())
		return obj, err
	}
	serverMetrics.UpdateDBRequestLatencyMetrics("etcd", "get", o.gvk.String(), time.Since(startTime))
	serverMetrics.UpdateDBRequestTotalMetrics("etcd", "get", o.gvk.String())
	klog.InfoS("get resp resp=%+v", resp)
	if len(resp.Kvs) != 1 {
		return obj, errors.NewNotFound(o.gr, key)
	}
	err = json.Unmarshal(resp.Kvs[0].Value, &obj)
	if err != nil {
		klog.ErrorS(err, "failed to marshal report kind=%s", o.gvk.String())
		return obj, errors.NewNotFound(o.gr, key)
	}
	return obj, nil
}

func (o *objectStoreNamespaced[T]) List(ctx context.Context, namespace string) ([]T, error) {
	o.Lock()
	defer o.Unlock()
	startTime := time.Now()
	var objects []T
	key := o.getPrefix(namespace)
	resp, err := o.etcdclient.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		klog.ErrorS(err, "failed to list report kind=%s", o.gvk.String())
		return objects, err
	}
	serverMetrics.UpdateDBRequestTotalMetrics("etcd", "list", o.gvk.String())
	serverMetrics.UpdateDBRequestLatencyMetrics("etcd", "list", o.gvk.String(), time.Since(startTime))
	klog.InfoS("list resp resp=%+v", resp)
	if len(resp.Kvs) == 0 {
		return objects, nil
	}
	objects = make([]T, 0, len(resp.Kvs))
	for _, v := range resp.Kvs {
		var obj T
		err = json.Unmarshal(v.Value, &obj)
		if err != nil {
			return objects, errors.NewNotFound(o.gr, key)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (o *objectStoreNamespaced[T]) Create(ctx context.Context, obj T) error {
	o.Lock()
	defer o.Unlock()
	startTime := time.Now()
	key := o.getKey(obj.GetName(), obj.GetNamespace())

	bObject, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// Upsert behavior - Put creates if not exists, updates if exists
	_, err = o.etcdclient.Put(ctx, key, string(bObject))
	if err != nil {
		klog.ErrorS(err, "failed to create report kind=%s", o.gvk.String())
		return err
	}

	serverMetrics.UpdateDBRequestTotalMetrics("etcd", "create", o.gvk.String())
	serverMetrics.UpdateDBRequestLatencyMetrics("etcd", "create", o.gvk.String(), time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("etcd", "create", obj, false)
	return nil
}

func (o *objectStoreNamespaced[T]) Update(ctx context.Context, obj T) error {
	o.Lock()
	defer o.Unlock()
	startTime := time.Now()
	key := o.getKey(obj.GetName(), obj.GetNamespace())

	bObject, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// Upsert behavior - Put creates if not exists, updates if exists
	_, err = o.etcdclient.Put(ctx, key, string(bObject))
	if err != nil {
		klog.ErrorS(err, "failed to update report kind=%s", o.gvk.String())
		return err
	}

	serverMetrics.UpdateDBRequestTotalMetrics("etcd", "update", o.gvk.String())
	serverMetrics.UpdateDBRequestLatencyMetrics("etcd", "update", o.gvk.String(), time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("etcd", "update", obj, false)
	return nil
}

func (o *objectStoreNamespaced[T]) Delete(ctx context.Context, name, namespace string) error {
	o.Lock()
	defer o.Unlock()

	startTime := time.Now()
	key := o.getKey(name, namespace)
	resp, err := o.etcdclient.Delete(ctx, key)
	if err != nil {
		klog.ErrorS(err, "failed to delete report kind=%s", o.gvk.String())
		return err
	}
	serverMetrics.UpdateDBRequestTotalMetrics("etcd", "delete", o.gvk.String())
	serverMetrics.UpdateDBRequestLatencyMetrics("etcd", "delete", o.gvk.String(), time.Since(startTime))
	if resp.Deleted == 0 {
		return errors.NewNotFound(o.gr, key)
	}
	return nil
}

type ObjectStorageCluster[T metav1.Object] interface {
	Get(ctx context.Context, name string) (T, error)
	List(ctx context.Context) ([]T, error)
	Create(ctx context.Context, obj T) error
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name string) error
}

type objectStoreCluster[T metav1.Object] struct {
	store ObjectStorageNamespaced[T]
}

func NewObjectStoreCluster[T metav1.Object](client clientv3.KV, gvk schema.GroupVersionKind, gr schema.GroupResource) ObjectStorageCluster[T] {
	return &objectStoreCluster[T]{
		store: &objectStoreNamespaced[T]{
			namespaced: false,
			etcdclient: client,
			gvk:        gvk,
			gr:         gr,
		},
	}
}

func (o *objectStoreCluster[T]) Get(ctx context.Context, name string) (T, error) {
	return o.store.Get(ctx, name, "")
}

func (o *objectStoreCluster[T]) List(ctx context.Context) ([]T, error) {
	return o.store.List(ctx, "")
}

func (o *objectStoreCluster[T]) Create(ctx context.Context, obj T) error {
	return o.store.Create(ctx, obj)
}

func (o *objectStoreCluster[T]) Update(ctx context.Context, obj T) error {
	return o.store.Update(ctx, obj)
}

func (o *objectStoreCluster[T]) Delete(ctx context.Context, name string) error {
	return o.store.Delete(ctx, name, "")
}
