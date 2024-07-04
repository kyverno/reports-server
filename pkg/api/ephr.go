package api

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
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
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
)

type ephrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func EphemeralReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &ephrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (p *ephrStore) New() runtime.Object {
	return &reportsv1.EphemeralReport{}
}

func (p *ephrStore) Destroy() {
}

func (p *ephrStore) Kind() string {
	return "EphemeralReport"
}

func (p *ephrStore) NewList() runtime.Object {
	return &reportsv1.EphemeralReportList{}
}

func (p *ephrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
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
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.Infof("listing ephemeral reports for namespace=%s", namespace)
	list, err := p.listEphr(namespace)
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource ephemeralreport")
	}

	// if labelSelector == labels.Everything() {
	// 	return list, nil
	// }

	ephrList := &reportsv1.EphemeralReportList{
		Items: make([]reportsv1.EphemeralReport, 0),
		ListMeta: metav1.ListMeta{
			// TODO: Fix this!!
			ResourceVersion: "1",
		},
	}
	for _, ephr := range list.Items {
		if ephr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(ephr.Labels)) {
			ephrList.Items = append(ephrList.Items, ephr)
		}
	}

	return ephrList, nil
}

func (p *ephrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.Infof("getting ephemeral reports name=%s namespace=%s", name, namespace)
	report, err := p.getEphr(name, namespace)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(utils.EphemeralReportsGR, name)
	}
	return report, nil
}

func (p *ephrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
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

	ephr, ok := obj.(*reportsv1.EphemeralReport)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate ephemeral report")
	}
	if ephr.Name == "" {
		if ephr.GenerateName == "" {
			return nil, errors.NewConflict(utils.EphemeralReportsGR, ephr.Name, fmt.Errorf("name and generate name not provided"))
		}
		ephr.Name = nameGenerator.GenerateName(ephr.GenerateName)
	}

	namespace := genericapirequest.NamespaceValue(ctx)

	if len(ephr.Namespace) == 0 {
		ephr.Namespace = namespace
	}

	ephr.Annotations = labelReports(ephr.Annotations)
	klog.Infof("creating ephemeral reports name=%s namespace=%s", ephr.Name, ephr.Namespace)
	if !isDryRun {
		r, err := p.createEphr(ephr)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot create ephemeral report: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Added, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (p *ephrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	oldObj, err := p.getEphr(name, namespace)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	ephr := updatedObject.(*reportsv1.EphemeralReport)

	if forceAllowCreate {
		r, err := p.updateEphr(ephr, oldObj)
		if err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := p.broadcaster.Action(watch.Modified, r); err != nil {
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

	ephr, ok := updatedObject.(*reportsv1.EphemeralReport)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate ephemeral report")
	}

	if len(ephr.Namespace) == 0 {
		ephr.Namespace = namespace
	}

	ephr.Annotations = labelReports(ephr.Annotations)
	klog.Infof("updating ephemeral reports name=%s namespace=%s", ephr.Name, ephr.Namespace)
	if !isDryRun {
		r, err := p.updateEphr(ephr, oldObj)
		if err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create ephemeral report: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Modified, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (p *ephrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	ephr, err := p.getEphr(name, namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to find ephrs", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewNotFound(utils.EphemeralReportsGR, name)
	}

	err = deleteValidation(ctx, ephr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting ephemeral reports name=%s namespace=%s", ephr.Name, ephr.Namespace)
	if !isDryRun {
		err = p.deleteEphr(ephr)
		if err != nil {
			klog.ErrorS(err, "failed to delete ephr", "name", name, "namespace", klog.KRef("", namespace))
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete ephemeralreport: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Deleted, ephr); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return ephr, true, nil // TODO: Add protobuf
}

func (p *ephrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	obj, err := p.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find ephrs", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to find ephemeral reports")
	}

	ephrList, ok := obj.(*reportsv1.EphemeralReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse ephrs", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to parse ephemeral reports")
	}

	if !isDryRun {
		for _, ephr := range ephrList.Items {
			_, isDeleted, err := p.Delete(ctx, ephr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete ephr", "name", ephr.GetName(), "namespace", klog.KRef("", namespace))
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete ephemeral report: %s/%s", ephr.Namespace, ephr.GetName()))
			}
		}
	}
	return ephrList, nil
}

func (p *ephrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return p.broadcaster.Watch()
}

func (p *ephrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *reportsv1.EphemeralReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addEphemeralReportToTable(&table, *t)
	case *reportsv1.EphemeralReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addEphemeralReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (p *ephrStore) NamespaceScoped() bool {
	return true
}

func (p *ephrStore) GetSingularName() string {
	return "ephemeralreport"
}

func (p *ephrStore) ShortNames() []string {
	return []string{"ephr"}
}

func (p *ephrStore) getEphr(name, namespace string) (*reportsv1.EphemeralReport, error) {
	val, err := p.store.EphemeralReports().Get(context.TODO(), name, namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find ephemeral report in store")
	}

	return val.DeepCopy(), nil
}

func (p *ephrStore) listEphr(namespace string) (*reportsv1.EphemeralReportList, error) {
	valList, err := p.store.EphemeralReports().List(context.TODO(), namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find ephemeral report in store")
	}

	reportList := &reportsv1.EphemeralReportList{
		Items: valList,
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (p *ephrStore) createEphr(report *reportsv1.EphemeralReport) (*reportsv1.EphemeralReport, error) {
	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return report, p.store.EphemeralReports().Create(context.TODO(), *report)
}

func (p *ephrStore) updateEphr(report *reportsv1.EphemeralReport, oldReport *reportsv1.EphemeralReport) (*reportsv1.EphemeralReport, error) {
	oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not parse resource version")
	}
	report.ResourceVersion = fmt.Sprint(oldRV + 1)

	return report, p.store.EphemeralReports().Update(context.TODO(), *report)
}

func (p *ephrStore) deleteEphr(report *reportsv1.EphemeralReport) error {
	return p.store.EphemeralReports().Delete(context.TODO(), report.Name, report.Namespace)
}
