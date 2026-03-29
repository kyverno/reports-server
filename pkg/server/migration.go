package server

import (
	"context"
	"sync"

	v1 "github.com/kyverno/kyverno/api/reports/v1"
	kyverno "github.com/kyverno/kyverno/pkg/clients/kyverno"
	"github.com/kyverno/reports-server/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/generated/v1alpha2/clientset/versioned"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	openreportsclient "github.com/openreports/reports-api/pkg/client/clientset/versioned/typed/openreports.io/v1alpha1"
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
	orClient, err := openreportsclient.NewForConfig(c.Rest)
	if err != nil {
		return err
	}
	workerChan := make(chan struct{}, numWorkers)

	if c.APIServices.StoreReports {
		if err := c.handleMigrateWgPolicyApis(ctx, policyClient, workerChan); err != nil {
			return err
		}
	}

	if c.APIServices.StoreEphemeralReports {
		if err := c.handleMigrateEphrApis(ctx, kyvernoClient, workerChan); err != nil {
			return err
		}
	}

	if c.APIServices.StoreOpenreports {
		if err := c.handleMigrateOpenreportsApis(ctx, orClient, workerChan); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) handleMigrateWgPolicyApis(ctx context.Context, policyClient *versioned.Clientset, workerChan chan struct{}) error {
	cpolrs, err := policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	count, err := c.Store.ClusterPolicyReports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.ClusterPolicyReportsGR)
		count = len(cpolrs.Items)
	}

	if count != len(cpolrs.Items) {
		if !c.SkipMigration {
			wg := &sync.WaitGroup{}
			wg.Add(len(cpolrs.Items))
			for _, r := range cpolrs.Items {
				applyReportsServerAnnotation(&r)
				go c.migrateReport(ctx, nil, policyClient, nil, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.ClusterPolicyReports().SetResourceVersion(cpolrs.ResourceVersion)
	if err != nil {
		return err
	}
	cpolrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Watch(ctx, options)
		},
	}
	cpolrWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, cpolrs.GetResourceVersion(), cpolrWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, cpolrWatchInterface)

	polrs, err := policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	count, err = c.Store.PolicyReports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.PolicyReportsGR)
		count = len(polrs.Items)
	}

	if count != len(polrs.Items) {
		if !c.SkipMigration || count != len(polrs.Items) {
			wg := &sync.WaitGroup{}
			wg.Add(len(polrs.Items))
			for _, r := range polrs.Items {
				applyReportsServerAnnotation(&r)
				go c.migrateReport(ctx, nil, policyClient, nil, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.PolicyReports().SetResourceVersion(polrs.ResourceVersion)
	if err != nil {
		return err
	}

	polrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").Watch(ctx, options)
		},
	}
	polrWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, polrs.GetResourceVersion(), polrWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, polrWatchInterface)
	return nil
}

func (c *Config) handleMigrateEphrApis(ctx context.Context, kyvernoClient kyverno.Interface, workerChan chan struct{}) error {
	cephrs, err := kyvernoClient.ReportsV1().ClusterEphemeralReports().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	count, err := c.Store.ClusterEphemeralReports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.ClusterEphemeralReportsGR)
		count = len(cephrs.Items)
	}

	if count != len(cephrs.Items) {
		if !c.SkipMigration {
			wg := &sync.WaitGroup{}
			wg.Add(len(cephrs.Items))
			for _, r := range cephrs.Items {
				applyReportsServerAnnotation(&r)
				go c.migrateReport(ctx, kyvernoClient, nil, nil, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.ClusterEphemeralReports().SetResourceVersion(cephrs.ResourceVersion)
	if err != nil {
		return err
	}
	cephrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kyvernoClient.ReportsV1().ClusterEphemeralReports().Watch(ctx, options)
		},
	}
	cephrWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, cephrs.GetResourceVersion(), cephrWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, cephrWatchInterface)

	ephrs, err := kyvernoClient.ReportsV1().EphemeralReports("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	count, err = c.Store.EphemeralReports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.EphemeralReportsGR)
		count = len(ephrs.Items)
	}

	if count != len(ephrs.Items) {
		if !c.SkipMigration {
			wg := &sync.WaitGroup{}
			wg.Add(len(ephrs.Items))
			for _, r := range ephrs.Items {
				applyReportsServerAnnotation(&r)
				// we don't the policyClient to be non nil because the ephemeral reports rely on kyvernoClient
				go c.migrateReport(ctx, kyvernoClient, nil, nil, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.EphemeralReports().SetResourceVersion(ephrs.ResourceVersion)
	if err != nil {
		return err
	}
	ephrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kyvernoClient.ReportsV1().EphemeralReports("").Watch(ctx, options)
		},
	}
	ephrWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, ephrs.GetResourceVersion(), ephrWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, ephrWatchInterface)
	return nil
}

func (c *Config) handleMigrateOpenreportsApis(ctx context.Context, orClient openreportsclient.OpenreportsV1alpha1Interface, workerChan chan struct{}) error {
	creps, err := orClient.ClusterReports().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	count, err := c.Store.ClusterReports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.OpenreportsClusterReportGR)
		count = len(creps.Items)
	}

	if count != len(creps.Items) {
		if !c.SkipMigration {
			wg := &sync.WaitGroup{}
			wg.Add(len(creps.Items))
			for _, r := range creps.Items {
				applyReportsServerAnnotation(&r)
				go c.migrateReport(ctx, nil, nil, orClient, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.ClusterReports().SetResourceVersion(creps.ResourceVersion)
	if err != nil {
		return err
	}
	crepsWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return orClient.ClusterReports().Watch(ctx, options)
		},
	}
	crepsWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, creps.GetResourceVersion(), crepsWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, crepsWatchInterface)

	reports, err := orClient.Reports("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	count, err = c.Store.Reports().Count(ctx)
	if err != nil {
		klog.Errorf("error listing the count of %s from the data store. falling back to the skip migration flag", utils.OpenreportsReportGR)
		count = len(reports.Items)
	}

	if count != len(reports.Items) {
		if !c.SkipMigration {
			wg := &sync.WaitGroup{}
			wg.Add(len(reports.Items))
			for _, r := range reports.Items {
				applyReportsServerAnnotation(&r)
				go c.migrateReport(ctx, nil, nil, orClient, workerChan, wg, r)
			}
			wg.Wait()
		}
	}

	err = c.Store.Reports().SetResourceVersion(reports.ResourceVersion)
	if err != nil {
		return err
	}

	reportsWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return orClient.Reports("").Watch(ctx, options)
		},
	}
	reportsWatchInterface, err := watchtools.NewRetryWatcherWithContext(ctx, reports.GetResourceVersion(), reportsWatcher)
	if err != nil {
		return err
	}
	go c.watchReport(ctx, reportsWatchInterface)
	return nil
}

func (c *Config) migrateReport(ctx context.Context,
	kyvernoClient kyverno.Interface,
	policyClient versioned.Interface,
	orClient openreportsclient.OpenreportsV1alpha1Interface,
	workerChan chan struct{},
	wg *sync.WaitGroup, report any) {
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
		if err := policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete cluster policy report %s: %s", r.Name, err)
		}
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
		if err := policyClient.Wgpolicyk8sV1alpha2().PolicyReports(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete policy report %s: %s", r.Name, err)
		}
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
		if err := kyvernoClient.ReportsV1().ClusterEphemeralReports().Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete cluster ephemeral report %s: %s", r.Name, err)
		}
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
		if err := kyvernoClient.ReportsV1().EphemeralReports(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete ephemeral report %s: %s", r.Name, err)
		}
	case openreportsv1alpha1.ClusterReport:
		err := c.Store.ClusterReports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = orClient.ClusterReports().Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		if err := orClient.ClusterReports().Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete cluster report %s: %s", r.Name, err)
		}
	case openreportsv1alpha1.Report:
		err := c.Store.Reports().Create(ctx, &r)
		if err != nil {
			klog.Errorf("failed to mirgrate report of kind %s %s: %s", r.GroupVersionKind().String(), r.Name, err)
		}
		// Update annotation in Kubernetes before deleting so watchers can identify it
		_, err = orClient.Reports(r.Namespace).Update(ctx, &r, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update annotation for report %s: %s", r.Name, err)
		}
		if err := orClient.Reports(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete report %s: %s", r.Name, err)
		}
	}
	<-workerChan
	// free the worker slot
}
