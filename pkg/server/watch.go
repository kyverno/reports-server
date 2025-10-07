package server

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/api"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

func (c *Config) watchEphr(ctx context.Context, watchIface *watchtools.RetryWatcher) {
	for {
		select {
		case <-ctx.Done():
		case event := <-watchIface.ResultChan():
			switch event.Type {
			case watch.Added:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				applyReportsServerAnnotation(ephr)
				err := c.Store.EphemeralReports().Create(ctx, ephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				applyReportsServerAnnotation(ephr)
				ephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := c.Store.EphemeralReports().Update(ctx, ephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				// Skip deletion if report was already migrated to the store
				if ephr.Annotations != nil && ephr.Annotations[api.ServedByReportsServerAnnotation] == api.ServedByReportsServerValue {
					continue
				}
				err := c.Store.EphemeralReports().Delete(ctx, ephr.Name, ephr.Namespace)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}
}

func (c *Config) watchCephr(ctx context.Context, watchIface *watchtools.RetryWatcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watchIface.ResultChan():
			switch event.Type {
			case watch.Added:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				applyReportsServerAnnotation(cephr)
				err := c.Store.ClusterEphemeralReports().Create(ctx, cephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				applyReportsServerAnnotation(cephr)
				err := c.Store.ClusterEphemeralReports().Update(ctx, cephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				// Skip deletion if report was already migrated to the store
				if cephr.Annotations != nil && cephr.Annotations[api.ServedByReportsServerAnnotation] == api.ServedByReportsServerValue {
					continue
				}
				err := c.Store.ClusterEphemeralReports().Delete(ctx, cephr.Name)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}
}

func (c *Config) watchPolr(ctx context.Context, watchIface *watchtools.RetryWatcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watchIface.ResultChan():
			switch event.Type {
			case watch.Added:
				polr := event.Object.(*v1alpha2.PolicyReport)
				applyReportsServerAnnotation(polr)
				err := c.Store.PolicyReports().Create(ctx, polr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				polr := event.Object.(*v1alpha2.PolicyReport)
				applyReportsServerAnnotation(polr)
				err := c.Store.PolicyReports().Update(ctx, polr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				polr := event.Object.(*v1alpha2.PolicyReport)
				// Skip deletion if report was already migrated to the store
				if polr.Annotations != nil && polr.Annotations[api.ServedByReportsServerAnnotation] == api.ServedByReportsServerValue {
					continue
				}
				err := c.Store.PolicyReports().Delete(ctx, polr.Name, polr.Namespace)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}
}

func (c *Config) watchCpolr(ctx context.Context, watchIface *watchtools.RetryWatcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watchIface.ResultChan():
			switch event.Type {
			case watch.Added:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				applyReportsServerAnnotation(cpolr)
				err := c.Store.ClusterPolicyReports().Create(ctx, cpolr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				applyReportsServerAnnotation(cpolr)
				err := c.Store.ClusterPolicyReports().Update(ctx, cpolr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				// Skip deletion if report was already migrated to the store
				if cpolr.Annotations != nil && cpolr.Annotations[api.ServedByReportsServerAnnotation] == api.ServedByReportsServerValue {
					continue
				}
				err := c.Store.ClusterPolicyReports().Delete(ctx, cpolr.Name)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}
}
