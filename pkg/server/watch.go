package server

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/api"

	storageapi "github.com/kyverno/reports-server/pkg/storage/api"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
)

func (c *Config) watchReport(ctx context.Context, watchIface *watchtools.RetryWatcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watchIface.ResultChan():
			switch event.Type {
			case watch.Added:
				handleCreate(ctx, c.Store, event.Object)
			case watch.Modified:
				handleUpdate(ctx, c.Store, event.Object)
			case watch.Deleted:
				handleDelete(ctx, c.Store, event.Object)
			}
		}
	}
}

func handleUpdate(ctx context.Context, storageIface storageapi.Storage, obj runtime.Object) {
	switch report := obj.(type) {
	case *v1alpha2.ClusterPolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterPolicyReports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *v1alpha2.PolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.PolicyReports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.ClusterEphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterEphemeralReports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.EphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.EphemeralReports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.Report:
		applyReportsServerAnnotation(report)
		err := storageIface.Reports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.ClusterReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterReports().Update(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	}
}

func handleCreate(ctx context.Context, storageIface storageapi.Storage, obj runtime.Object) {
	switch report := obj.(type) {
	case *v1alpha2.ClusterPolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterPolicyReports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *v1alpha2.PolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.PolicyReports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.ClusterEphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterEphemeralReports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.EphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.EphemeralReports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.Report:
		applyReportsServerAnnotation(report)
		err := storageIface.Reports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.ClusterReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterReports().Create(ctx, report)
		if err != nil {
			klog.Error(err)
		}
	}
}

func handleDelete(ctx context.Context, storageIface storageapi.Storage, obj runtime.Object) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	annotations := accessor.GetAnnotations()
	if annotations != nil && annotations[api.ServedByReportsServerAnnotation] == api.ServedByReportsServerValue {
		return
	}

	switch report := obj.(type) {
	case *v1alpha2.ClusterPolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterPolicyReports().Delete(ctx, report.Name)
		if err != nil {
			klog.Error(err)
		}
	case *v1alpha2.PolicyReport:
		applyReportsServerAnnotation(report)
		err := storageIface.PolicyReports().Delete(ctx, report.Name, report.Namespace)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.ClusterEphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterEphemeralReports().Delete(ctx, report.Name)
		if err != nil {
			klog.Error(err)
		}
	case *reportsv1.EphemeralReport:
		applyReportsServerAnnotation(report)
		err := storageIface.EphemeralReports().Delete(ctx, report.Name, report.Namespace)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.Report:
		applyReportsServerAnnotation(report)
		err := storageIface.Reports().Delete(ctx, report.Name, report.Namespace)
		if err != nil {
			klog.Error(err)
		}
	case *openreportsv1alpha1.ClusterReport:
		applyReportsServerAnnotation(report)
		err := storageIface.ClusterReports().Delete(ctx, report.Name)
		if err != nil {
			klog.Error(err)
		}
	}
}
