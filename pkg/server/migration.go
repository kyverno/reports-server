package server

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	v1 "github.com/kyverno/kyverno/api/reports/v1"
	kyverno "github.com/kyverno/kyverno/pkg/clients/kyverno"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/generated/v1alpha2/clientset/versioned"
)

func (c *Config) migration(ctx context.Context) error {
	kyvernoClient, err := kyverno.NewForConfig(c.Rest)
	if err != nil {
		return err
	}
	policyClient, err := versioned.NewForConfig(c.Rest)
	if err != nil {
		return err
	}
	workerChan := make(chan struct{}, numWorkers)

	if c.APIServices.StoreReports {
		cpolrs, err := policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}

		cpolrWg := &sync.WaitGroup{}
		cpolrWg.Add(len(cpolrs.Items))
		for _, r := range cpolrs.Items {
			applyReportsServerAnnotation(&r)
			go c.migrateReport(ctx, kyvernoClient, policyClient, workerChan, cpolrWg, r)
		}
		cpolrWg.Wait()

		err = c.Store.SetResourceVersion(cpolrs.ResourceVersion)
		if err != nil {
			return err
		}
		cpolrWatcher := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Watch(ctx, options)
			},
		}
		cpolrWatchInterface, err := watchtools.NewRetryWatcher(cpolrs.GetResourceVersion(), cpolrWatcher)
		if err != nil {
			return err
		}
		go c.watchCpolr(ctx, cpolrWatchInterface)

		polrs, err := policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}

		polrWg := &sync.WaitGroup{}
		polrWg.Add(len(polrs.Items))
		for _, r := range polrs.Items {
			applyReportsServerAnnotation(&r)
			go c.migrateReport(ctx, kyvernoClient, policyClient, workerChan, polrWg, r)
		}
		polrWg.Wait()

		err = c.Store.SetResourceVersion(polrs.ResourceVersion)
		if err != nil {
			return err
		}

		polrWatcher := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").Watch(ctx, options)
			},
		}
		polrWatchInterface, err := watchtools.NewRetryWatcher(polrs.GetResourceVersion(), polrWatcher)
		if err != nil {
			return err
		}
		go c.watchPolr(ctx, polrWatchInterface)
	}

	if c.APIServices.StoreEphemeralReports {
		cephrs, err := kyvernoClient.ReportsV1().ClusterEphemeralReports().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}

		cephrWg := &sync.WaitGroup{}
		cephrWg.Add(len(cephrs.Items))
		for _, r := range cephrs.Items {
			applyReportsServerAnnotation(&r)
			go c.migrateReport(ctx, kyvernoClient, policyClient, workerChan, cephrWg, r)
		}
		cephrWg.Wait()

		err = c.Store.SetResourceVersion(cephrs.ResourceVersion)
		if err != nil {
			return err
		}
		cephrWatcher := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kyvernoClient.ReportsV1().ClusterEphemeralReports().Watch(ctx, options)
			},
		}
		cephrWatchInterface, err := watchtools.NewRetryWatcher(cephrs.GetResourceVersion(), cephrWatcher)
		if err != nil {
			return err
		}
		go c.watchCephr(ctx, cephrWatchInterface)

		ephrs, err := kyvernoClient.ReportsV1().EphemeralReports("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}
		ephrWg := &sync.WaitGroup{}
		ephrWg.Add(len(ephrs.Items))
		for _, r := range ephrs.Items {
			applyReportsServerAnnotation(&r)
			go c.migrateReport(ctx, kyvernoClient, policyClient, workerChan, ephrWg, r)
		}
		ephrWg.Wait()

		err = c.Store.SetResourceVersion(ephrs.ResourceVersion)
		if err != nil {
			return err
		}
		ephrWatcher := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kyvernoClient.ReportsV1().EphemeralReports("").Watch(ctx, options)
			},
		}
		ephrWatchInterface, err := watchtools.NewRetryWatcher(ephrs.GetResourceVersion(), ephrWatcher)
		if err != nil {
			return err
		}
		go c.watchEphr(ctx, ephrWatchInterface)
	}
	rv, err := strconv.ParseUint(c.Store.UseResourceVersion(), 10, 64)
	if err != nil {
		return err
	}
	// use leave some versions for resources added using watchers
	return c.Store.SetResourceVersion(fmt.Sprint((rv + 9999)))
}

func (c *Config) migrateReport(ctx context.Context, kyvernoClient kyverno.Interface, policyClient versioned.Interface, workerChan chan struct{}, wg *sync.WaitGroup, report interface{}) {
	defer wg.Done()

	// book a slot in the worker chan
	workerChan <- struct{}{}
	switch r := report.(type) {
	case v1alpha2.ClusterPolicyReport:
		err := c.Store.ClusterPolicyReports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Delete(ctx, r.Name, metav1.DeleteOptions{})
	case v1alpha2.PolicyReport:
		err := c.Store.PolicyReports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = policyClient.Wgpolicyk8sV1alpha2().PolicyReports(r.Namespace).Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		policyClient.Wgpolicyk8sV1alpha2().PolicyReports(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{})
	case v1.ClusterEphemeralReport:
		err := c.Store.ClusterEphemeralReports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = kyvernoClient.ReportsV1().ClusterEphemeralReports().Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		kyvernoClient.ReportsV1().ClusterEphemeralReports().Delete(ctx, r.Name, metav1.DeleteOptions{})
	case v1.EphemeralReport:
		err := c.Store.EphemeralReports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = kyvernoClient.ReportsV1().EphemeralReports(r.Namespace).Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		kyvernoClient.ReportsV1().EphemeralReports(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{})
	}
	<-workerChan
	// free the worker slot
}
