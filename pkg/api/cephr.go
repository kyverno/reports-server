package api

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/nirmata/reports-server/pkg/storage"
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
)

type cephrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func ClusterEphemeralReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &cephrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (c *cephrStore) New() runtime.Object {
	return &reportsv1.ClusterEphemeralReport{}
}

func (c *cephrStore) Destroy() {
}

func (c *cephrStore) Kind() string {
	return "ClusterEphemeralReport"
}

func (c *cephrStore) NewList() runtime.Object {
	return &reportsv1.ClusterEphemeralReportList{}
}

func (c *cephrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
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
	klog.Infof("listing cluster ephemeral reports")
	list, err := c.listCephr()
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource clusterephemeralreport")
	}

	// if labelSelector.String() == labels.Everything().String() {
	// 	return list, nil
	// }

	cephrList := &reportsv1.ClusterEphemeralReportList{
		Items: make([]reportsv1.ClusterEphemeralReport, 0),
	}
	for _, cephr := range list.Items {
		if cephr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(cephr.Labels)) {
			cephrList.Items = append(cephrList.Items, cephr)
		}
	}

	return cephrList, nil
}

func (c *cephrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	klog.Infof("getting cluster ephemeral reports name=%s", name)
	report, err := c.getCephr(name)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(reportsv1.Resource("clusterephemeralreports"), name)
	}
	return report, nil
}

func (c *cephrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
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

	cephr, ok := obj.(*reportsv1.ClusterEphemeralReport)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate cluster ephemeral report")
	}

	klog.Infof("creating cluster ephemeral reports name=%s", cephr.Name)
	if !isDryRun {
		if err := c.createCephr(cephr); err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot create cluster ephemeral report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Added, obj); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (c *cephrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	oldObj, err := c.getCephr(name)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	cephr := updatedObject.(*reportsv1.ClusterEphemeralReport)
	if forceAllowCreate {
		if err := c.updateCephr(cephr, oldObj); err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := c.broadcaster.Action(watch.Added, updatedObject); err != nil {
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

	cephr, ok := updatedObject.(*reportsv1.ClusterEphemeralReport)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate cluster ephemeral report")
	}

	klog.Infof("updating cluster ephemeral reports name=%s", cephr.Name)
	if !isDryRun {
		if err := c.updateCephr(cephr, oldObj); err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create cluster ephemeral report: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Modified, updatedObject); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (c *cephrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	cephr, err := c.getCephr(name)
	if err != nil {
		klog.ErrorS(err, "Failed to find cephrs", "name", name)
		return nil, false, errors.NewNotFound(reportsv1.Resource("clusterephemeralreports"), name)
	}

	err = deleteValidation(ctx, cephr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name)
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting cluster ephemeral reports name=%s", cephr.Name)
	if !isDryRun {
		if err = c.deleteCephr(cephr); err != nil {
			klog.ErrorS(err, "failed to delete cephr", "name", name)
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete clusterephemeralreport: %s", err.Error()))
		}
		if err := c.broadcaster.Action(watch.Deleted, cephr); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return cephr, true, nil // TODO: Add protobuf
}

func (c *cephrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	obj, err := c.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find cephrs")
		return nil, errors.NewBadRequest("Failed to find cluster ephemeral reports")
	}

	cephrList, ok := obj.(*reportsv1.ClusterEphemeralReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse cephrs")
		return nil, errors.NewBadRequest("Failed to parse cluster ephemeral reports")
	}

	if !isDryRun {
		for _, cephr := range cephrList.Items {
			obj, isDeleted, err := c.Delete(ctx, cephr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete cephr", "name", cephr.GetName())
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete cluster ephemral report: %s", cephr.GetName()))
			}
			if err := c.broadcaster.Action(watch.Deleted, obj); err != nil {
				klog.ErrorS(err, "failed to broadcast event")
			}
		}
	}
	return cephrList, nil
}

func (c *cephrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return c.broadcaster.Watch()
}

func (c *cephrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *reportsv1.ClusterEphemeralReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addClusterEphemeralReportToTable(&table, *t)
	case *reportsv1.ClusterEphemeralReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addClusterEphemeralReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (c *cephrStore) NamespaceScoped() bool {
	return false
}

func (c *cephrStore) GetSingularName() string {
	return "clusterephemeralreport"
}

func (c *cephrStore) ShortNames() []string {
	return []string{"cephr"}
}

func (c *cephrStore) getCephr(name string) (*reportsv1.ClusterEphemeralReport, error) {
	val, err := c.store.ClusterEphemeralReports().Get(context.TODO(), name)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster ephemeral report in store")
	}

	return val.DeepCopy(), nil
}

func (c *cephrStore) listCephr() (*reportsv1.ClusterEphemeralReportList, error) {
	valList, err := c.store.ClusterEphemeralReports().List(context.TODO())
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster ephemeral report in store")
	}

	reportList := &reportsv1.ClusterEphemeralReportList{
		Items: valList,
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (c *cephrStore) createCephr(report *reportsv1.ClusterEphemeralReport) error {
	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return c.store.ClusterEphemeralReports().Create(context.TODO(), *report)
}

func (c *cephrStore) updateCephr(report *reportsv1.ClusterEphemeralReport, oldReport *reportsv1.ClusterEphemeralReport) error {
	oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
	if err != nil {
		return errorpkg.Wrapf(err, "could not parse resource version")
	}
	report.ResourceVersion = fmt.Sprint(oldRV + 1)

	return c.store.ClusterEphemeralReports().Update(context.TODO(), *report)
}

func (c *cephrStore) deleteCephr(report *reportsv1.ClusterEphemeralReport) error {
	return c.store.ClusterEphemeralReports().Delete(context.TODO(), report.Name)
}
