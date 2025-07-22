package api

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/kyverno/reports-server/pkg/storage"
	"github.com/kyverno/reports-server/pkg/utils"
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
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type clusterReportStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func ClusterReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &clusterReportStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (c *clusterReportStore) New() runtime.Object {
	return &openreportsv1alpha1.ClusterReportList{}
}

func (c *clusterReportStore) Destroy() {
}

func (c *clusterReportStore) Kind() string {
	return "ClusterReport"
}

func (c *clusterReportStore) NewList() runtime.Object {
	return &openreportsv1alpha1.ClusterReportList{}
}

func (c *clusterReportStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	var labelSelector labels.Selector
	if options != nil {
		if options.LabelSelector != nil {
			labelSelector = options.LabelSelector
		}
	}
	klog.Infof("listing all cluster policy reports")
	list, err := c.listCpolr()
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource clusterpolicyreport")
	}

	cpolrList := &openreportsv1alpha1.ClusterReportList{
		Items:    make([]openreportsv1alpha1.ClusterReport, 0),
		ListMeta: metav1.ListMeta{},
	}
	var desiredRv uint64
	if len(options.ResourceVersion) == 0 {
		desiredRv = 1
	} else {
		desiredRv, err = strconv.ParseUint(options.ResourceVersion, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	var resourceVersion uint64
	resourceVersion = 1
	for _, cpolr := range list.Items {
		allow, rv, err := allowObjectListWatch(cpolr.ObjectMeta, labelSelector, desiredRv, options.ResourceVersionMatch)
		if err != nil {
			return nil, err
		}
		if rv > resourceVersion {
			resourceVersion = rv
		}
		if allow {
			cpolrList.Items = append(cpolrList.Items, cpolr)
		}
	}
	cpolrList.ListMeta.ResourceVersion = strconv.FormatUint(resourceVersion, 10)
	klog.Infof("filtered list found length: %d", len(cpolrList.Items))
	return cpolrList, nil
}

func (c *clusterReportStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	klog.Infof("fetching cluster report name=%s", name)
	report, err := c.getCpolr(name)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(utils.OpenreportsClusterReportGR, name)
	}
	return report, nil
}

func (c *clusterReportStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	err := createValidation(ctx, obj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
		case "Strict":
			return nil, err
		}
	}

	cpolr, ok := obj.(*openreportsv1alpha1.ClusterReport)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate cluster policy report")
	}
	if cpolr.Name == "" {
		if cpolr.GenerateName == "" {
			return nil, errors.NewAlreadyExists(utils.OpenreportsClusterReportGR, cpolr.Name)
		}
		cpolr.Name = nameGenerator.GenerateName(cpolr.GenerateName)
	}

	cpolr.Annotations = labelReports(cpolr.Annotations)
	cpolr.Generation = 1
	klog.Infof("creating cluster report name=%s", cpolr.Name)
	if !isDryRun {
		r, err := c.createCpolr(cpolr)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Added, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (c *clusterReportStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	oldObj, err := c.getCpolr(name)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	cpolr := updatedObject.(*openreportsv1alpha1.ClusterReport)
	if forceAllowCreate {
		r, err := c.updateCpolr(cpolr, oldObj)
		if err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := c.broadcaster.Action(watch.Modified, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
		return updatedObject, true, nil
	}

	err = updateValidation(ctx, updatedObject, oldObj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
		case "Strict":
			return nil, false, err
		}
	}

	cpolr, ok := updatedObject.(*openreportsv1alpha1.ClusterReport)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate cluster policy report")
	}
	cpolr.Annotations = labelReports(cpolr.Annotations)
	cpolr.Generation += 1
	klog.Infof("updating cluster report name=%s", cpolr.Name)
	if !isDryRun {
		r, err := c.updateCpolr(cpolr, oldObj)
		if err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Modified, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (c *clusterReportStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	cpolr, err := c.getCpolr(name)
	if err != nil {
		klog.ErrorS(err, "Failed to find creps", "name", name)
		return nil, false, errors.NewNotFound(utils.OpenreportsClusterReportGR, name)
	}

	err = deleteValidation(ctx, cpolr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name)
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting cluster report name=%s", cpolr.Name)
	if !isDryRun {
		if err = c.deleteCpolr(cpolr); err != nil {
			klog.ErrorS(err, "failed to delete cpolr", "name", name)
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete clusterpolicyreport: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Deleted, cpolr); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return cpolr, true, nil
}

func (c *clusterReportStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	obj, err := c.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find creps")
		return nil, errors.NewBadRequest("Failed to find cluster policy reports")
	}

	cpolrList, ok := obj.(*openreportsv1alpha1.ClusterReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse creps")
		return nil, errors.NewBadRequest("Failed to parse cluster policy reports")
	}

	if !isDryRun {
		for _, cpolr := range cpolrList.Items {
			_, isDeleted, err := c.Delete(ctx, cpolr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete cpolr", "name", cpolr.GetName())
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete cluster policy report: %s", cpolr.GetName()))
			}
		}
	}
	return cpolrList, nil
}

func (c *clusterReportStore) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	klog.Infof("watching cluster policy reports rv=%s", options.ResourceVersion)
	switch options.ResourceVersion {
	case "", "0":
		return c.broadcaster.Watch()
	default:
		break
	}
	items, err := c.List(ctx, options)
	if err != nil {
		return nil, err
	}
	list, ok := items.(*openreportsv1alpha1.ClusterReportList)
	if !ok {
		return nil, fmt.Errorf("failed to convert runtime object into cluster report list")
	}
	events := make([]watch.Event, len(list.Items))
	for i, pol := range list.Items {
		report := pol.DeepCopy()
		events[i] = watch.Event{
			Type:   watch.Added,
			Object: report,
		}
	}
	return c.broadcaster.WatchWithPrefix(events)
}

func (c *clusterReportStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *openreportsv1alpha1.ClusterReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addOpenreportsClusterReportToTable(&table, *t)
	case *openreportsv1alpha1.ClusterReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addOpenreportsClusterReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (c *clusterReportStore) NamespaceScoped() bool {
	return false
}

func (c *clusterReportStore) GetSingularName() string {
	return "clusterreport"
}

func (c *clusterReportStore) ShortNames() []string {
	return []string{"creps"}
}

func (c *clusterReportStore) getCpolr(name string) (*openreportsv1alpha1.ClusterReport, error) {
	val, err := c.store.ClusterReports().Get(context.TODO(), name)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster report in store")
	}

	return val, nil
}

func (c *clusterReportStore) listCpolr() (*openreportsv1alpha1.ClusterReportList, error) {
	valList, err := c.store.ClusterReports().List(context.TODO())
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster report in store")
	}

	reportList := &openreportsv1alpha1.ClusterReportList{
		Items: make([]openreportsv1alpha1.ClusterReport, 0, len(valList)),
	}

	for _, v := range valList {
		reportList.Items = append(reportList.Items, *v.DeepCopy())
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (c *clusterReportStore) createCpolr(report *openreportsv1alpha1.ClusterReport) (*openreportsv1alpha1.ClusterReport, error) {
	report.ResourceVersion = c.store.UseResourceVersion()
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return report, c.store.ClusterReports().Create(context.TODO(), report)
}

func (c *clusterReportStore) updateCpolr(report *openreportsv1alpha1.ClusterReport, _ *openreportsv1alpha1.ClusterReport) (*openreportsv1alpha1.ClusterReport, error) {
	report.ResourceVersion = c.store.UseResourceVersion()
	return report, c.store.ClusterReports().Update(context.TODO(), report)
}

func (c *clusterReportStore) deleteCpolr(report *openreportsv1alpha1.ClusterReport) error {
	return c.store.ClusterReports().Delete(context.TODO(), report.Name)
}
