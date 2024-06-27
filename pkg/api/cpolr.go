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
	klog.Infof("listing all cluster policy reports")
	list, err := c.listCpolr()
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource clusterpolicyreport")
	}

	// if labelSelector.String() == labels.Everything().String() {
	// 	return list, nil
	// }

	cpolrList := &v1alpha2.ClusterPolicyReportList{
		Items: make([]v1alpha2.ClusterPolicyReport, 0),
		ListMeta: metav1.ListMeta{
			// TODO: Fix this!!
			ResourceVersion: "1",
		},
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
	klog.Infof("fetching cluster policy report name=%s", name)
	report, err := c.getCpolr(name)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(utils.ClusterPolicyReportsGR, name)
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
	if cpolr.Name == "" {
		if cpolr.GenerateName == "" {
			return nil, errors.NewConflict(utils.ClusterPolicyReportsGR, cpolr.Name, fmt.Errorf("name and generate name not provided"))
		}
		cpolr.Name = nameGenerator.GenerateName(cpolr.GenerateName)
	}

	klog.Infof("creating cluster policy report name=%s", cpolr.Name)
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

func (c *cpolrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	oldObj, err := c.getCpolr(name)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	cpolr := updatedObject.(*v1alpha2.ClusterPolicyReport)
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

	klog.Infof("updating cluster policy report name=%s", cpolr.Name)
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

func (c *cpolrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	cpolr, err := c.getCpolr(name)
	if err != nil {
		klog.ErrorS(err, "Failed to find cpolrs", "name", name)
		return nil, false, errors.NewNotFound(utils.ClusterPolicyReportsGR, name)
	}

	err = deleteValidation(ctx, cpolr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name)
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting cluster policy report name=%s", cpolr.Name)
	if !isDryRun {
		if err = c.deleteCpolr(cpolr); err != nil {
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
			_, isDeleted, err := c.Delete(ctx, cpolr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete cpolr", "name", cpolr.GetName())
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete cluster policy report: %s", cpolr.GetName()))
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

func (c *cpolrStore) createCpolr(report *v1alpha2.ClusterPolicyReport) (*v1alpha2.ClusterPolicyReport, error) {
	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return report, c.store.ClusterPolicyReports().Create(context.TODO(), *report)
}

func (c *cpolrStore) updateCpolr(report *v1alpha2.ClusterPolicyReport, oldReport *v1alpha2.ClusterPolicyReport) (*v1alpha2.ClusterPolicyReport, error) {
	oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not parse resource version")
	}
	report.ResourceVersion = fmt.Sprint(oldRV + 1)

	return report, c.store.ClusterPolicyReports().Update(context.TODO(), *report)
}

func (c *cpolrStore) deleteCpolr(report *v1alpha2.ClusterPolicyReport) error {
	return c.store.ClusterPolicyReports().Delete(context.TODO(), report.Name)
}
