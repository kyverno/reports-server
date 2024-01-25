package api

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/kyverno/reports-server/pkg/storage"
	errorpkg "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type cpolrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func ClusterPolicyReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &cpolrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (c *cpolrStore) New() runtime.Object {
	return &v1alpha2.ClusterPolicyReport{}
}

func (c *cpolrStore) Destroy() {
}

func (c *cpolrStore) Kind() string {
	return "ClusterPolicyReport"
}

func (c *cpolrStore) NewList() runtime.Object {
	return &v1alpha2.ClusterPolicyReportList{}
}

func (c *cpolrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	labelSelector := labels.Everything()
	// fieldSelector := fields.Everything() // TODO: Field selectors
	if options != nil {
		if options.LabelSelector != nil {
			labelSelector = options.LabelSelector
		}
		// if options.FieldSelector != nil {
		// 	fieldSelector = options.FieldSelector
		// }
	}
	list, err := c.listCpolr()
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource clusterpolicyreport")
	}

	// if labelSelector.String() == labels.Everything().String() {
	// 	return list, nil
	// }

	cpolrList := &v1alpha2.ClusterPolicyReportList{
		Items: make([]v1alpha2.ClusterPolicyReport, 0),
	}
	for _, cpolr := range list.Items {
		if cpolr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(cpolr.Labels)) {
			cpolrList.Items = append(cpolrList.Items, cpolr)
		}
	}

	return cpolrList, nil
}

func (c *cpolrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	report, err := c.getCpolr(name)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(v1alpha2.Resource("clusterpolicyreports"), name)
	}
	return report, nil
}

func (c *cpolrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	err := createValidation(ctx, obj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
			// return &admissionv1.AdmissionResponse{
			// 	Allowed:  false,
			// 	Warnings: []string{err.Error()},
			// }, nil
		case "Strict":
			return nil, err
		}
	}

	cpolr, ok := obj.(*v1alpha2.ClusterPolicyReport)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate cluster policy report")
	}

	if !isDryRun {
		if err := c.createCpolr(cpolr); err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Added, obj); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (c *cpolrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	if forceAllowCreate {
		oldObj, _ := c.getCpolr(name)
		updatedObject, _ := objInfo.UpdatedObject(ctx, oldObj)
		cpolr := updatedObject.(*v1alpha2.ClusterPolicyReport)
		if err := c.updatePolr(cpolr, true); err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := c.broadcaster.Action(watch.Added, updatedObject); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
		return updatedObject, true, nil
	}

	oldObj, err := c.getCpolr(name)
	if err != nil {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	err = updateValidation(ctx, updatedObject, oldObj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
			// return &admissionv1.AdmissionResponse{
			// 	Allowed:  false,
			// 	Warnings: []string{err.Error()},
			// }, nil
		case "Strict":
			return nil, false, err
		}
	}

	cpolr, ok := updatedObject.(*v1alpha2.ClusterPolicyReport)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate cluster policy report")
	}

	if !isDryRun {
		if err := c.createCpolr(cpolr); err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Modified, updatedObject); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (c *cpolrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	cpolr, err := c.getCpolr(name)
	if err != nil {
		klog.ErrorS(err, "Failed to find cpolrs", "name", name)
		return nil, false, errors.NewNotFound(v1alpha2.Resource("clusterpolicyreports"), name)
	}

	err = deleteValidation(ctx, cpolr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name)
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	if !isDryRun {
		if err = c.deletePolr(cpolr); err != nil {
			klog.ErrorS(err, "failed to delete cpolr", "name", name)
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete clusterpolicyreport: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Deleted, cpolr); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return cpolr, true, nil // TODO: Add protobuf in wgpolicygroup
}

func (c *cpolrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	obj, err := c.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find cpolrs")
		return nil, errors.NewBadRequest("Failed to find cluster policy reports")
	}

	cpolrList, ok := obj.(*v1alpha2.ClusterPolicyReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse cpolrs")
		return nil, errors.NewBadRequest("Failed to parse cluster policy reports")
	}

	if !isDryRun {
		for _, cpolr := range cpolrList.Items {
			obj, isDeleted, err := c.Delete(ctx, cpolr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete cpolr", "name", cpolr.GetName())
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete cluster policy report: %s", cpolr.GetName()))
			}
			if err := c.broadcaster.Action(watch.Deleted, obj); err != nil {
				klog.ErrorS(err, "failed to broadcast event")
			}
		}
	}
	return cpolrList, nil
}

func (c *cpolrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return c.broadcaster.Watch()
}

func (c *cpolrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1alpha2.ClusterPolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addClusterPolicyReportToTable(&table, *t)
	case *v1alpha2.ClusterPolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addClusterPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (c *cpolrStore) NamespaceScoped() bool {
	return false
}

func (c *cpolrStore) GetSingularName() string {
	return "clusterpolicyreport"
}

func (c *cpolrStore) ShortNames() []string {
	return []string{"cpolr"}
}

func (c *cpolrStore) getCpolr(name string) (*v1alpha2.ClusterPolicyReport, error) {
	val, err := c.store.ClusterPolicyReports().Get(context.TODO(), name)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster policy report in store")
	}

	return val.DeepCopy(), nil
}

func (c *cpolrStore) listCpolr() (*v1alpha2.ClusterPolicyReportList, error) {
	valList, err := c.store.ClusterPolicyReports().List(context.TODO())
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster policy report in store")
	}

	reportList := &v1alpha2.ClusterPolicyReportList{
		Items: valList,
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (c *cpolrStore) createCpolr(report *v1alpha2.ClusterPolicyReport) error {
	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return c.store.ClusterPolicyReports().Create(context.TODO(), *report)
}

func (c *cpolrStore) updatePolr(report *v1alpha2.ClusterPolicyReport, force bool) error {
	if !force {
		oldReport, err := c.getCpolr(report.GetName())
		if err != nil {
			return errorpkg.Wrapf(err, "old cluster policy report not found")
		}
		oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
		if err != nil {
			return errorpkg.Wrapf(err, "could not parse resource version")
		}

		report.ResourceVersion = fmt.Sprint(oldRV + 1)
	} else {
		report.ResourceVersion = "1"
	}

	return c.store.ClusterPolicyReports().Update(context.TODO(), *report)
}

func (c *cpolrStore) deletePolr(report *v1alpha2.ClusterPolicyReport) error {
	return c.store.ClusterPolicyReports().Delete(context.TODO(), report.Name)
}
