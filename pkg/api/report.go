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
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type reportStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func ReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &reportStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (s *reportStore) New() runtime.Object {
	return &openreportsv1alpha1.Report{}
}

func (s *reportStore) Destroy() {
}

func (s *reportStore) Kind() string {
	return "Report"
}

func (s *reportStore) NewList() runtime.Object {
	return &openreportsv1alpha1.ReportList{}
}

func (s *reportStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	var labelSelector labels.Selector
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

	klog.Infof("listing reports for namespace=%s", namespace)
	list, err := s.listRep(namespace)
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource policyreport")
	}

	// if labelSelector == labels.Everything() {
	// 	return list, nil
	// }

	repList := &openreportsv1alpha1.ReportList{
		Items:    make([]openreportsv1alpha1.Report, 0),
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
	for _, rep := range list.Items {
		allow, rv, err := allowObjectListWatch(rep.ObjectMeta, labelSelector, desiredRv, options.ResourceVersionMatch)
		if err != nil {
			return nil, err
		}
		if rv > resourceVersion {
			resourceVersion = rv
		}
		if allow {
			repList.Items = append(repList.Items, rep)
		}
	}
	repList.ListMeta.ResourceVersion = strconv.FormatUint(resourceVersion, 10)
	klog.Infof("filtered list found length: %d", len(repList.Items))
	return repList, nil
}

func (s *reportStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.Infof("getting reports name=%s namespace=%s", name, namespace)
	report, err := s.getRep(name, namespace)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(utils.PolicyReportsGR, name)
	}
	return report, nil
}

func (s *reportStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	err := createValidation(ctx, obj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
			// return &admissionv1.AdmissionResponse{
			// 	Allowed:  false,
			// 	Warnings: []string{ers.Error()},
			// }, nil
		case "Strict":
			return nil, err
		}
	}

	rep, ok := obj.(*openreportsv1alpha1.Report)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate report")
	}

	namespace := genericapirequest.NamespaceValue(ctx)

	if len(rep.Namespace) == 0 {
		rep.Namespace = namespace
	}
	if rep.Name == "" {
		if rep.GenerateName == "" {
			return nil, errors.NewConflict(utils.PolicyReportsGR, rep.Name, fmt.Errorf("name and generate name not provided"))
		}
		rep.Name = nameGenerator.GenerateName(rep.GenerateName)
	}

	rep.Annotations = labelReports(rep.Annotations)
	rep.Generation = 1
	klog.Infof("creating reports name=%s namespace=%s", rep.Name, rep.Namespace)
	if !isDryRun {
		r, err := s.createRep(rep)
		if err != nil {
			return nil, errors.NewAlreadyExists(utils.PolicyReportsGR, rep.Name)
		}
		if err := s.broadcaster.Action(watch.Added, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (s *reportStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	oldObj, err := s.getRep(name, namespace)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	rep := updatedObject.(*openreportsv1alpha1.Report)

	if forceAllowCreate {
		r, err := s.updateRep(rep, oldObj)
		if err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := s.broadcaster.Action(watch.Modified, r); err != nil {
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
			// 	Warnings: []string{ers.Error()},
			// }, nil
		case "Strict":
			return nil, false, err
		}
	}

	rep, ok := updatedObject.(*openreportsv1alpha1.Report)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate report")
	}

	if len(rep.Namespace) == 0 {
		rep.Namespace = namespace
	}

	rep.Annotations = labelReports(rep.Annotations)
	rep.Generation += 1
	klog.Infof("updating reports name=%s namespace=%s", rep.Name, rep.Namespace)
	if !isDryRun {
		r, err := s.updateRep(rep, oldObj)
		if err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create report: %s", err.Error()))
		}
		if err := s.broadcaster.Action(watch.Modified, r); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (s *reportStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	rep, err := s.getRep(name, namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to find reports", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewNotFound(utils.PolicyReportsGR, name)
	}

	err = deleteValidation(ctx, rep)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting reports name=%s namespace=%s", rep.Name, rep.Namespace)
	if !isDryRun {
		err = s.deleteRep(rep)
		if err != nil {
			klog.ErrorS(err, "failed to delete reports", "name", name, "namespace", klog.KRef("", namespace))
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete policyreport: %s", err.Error()))
		}
		if err := s.broadcaster.Action(watch.Deleted, rep); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return rep, true, nil
}

func (s *reportStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	obj, err := s.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find reports", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to find reports")
	}

	RepList, ok := obj.(*openreportsv1alpha1.ReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse reports", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to parse reports")
	}

	if !isDryRun {
		for _, rep := range RepList.Items {
			_, isDeleted, err := s.Delete(ctx, rep.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete reports", "name", rep.GetName(), "namespace", klog.KRef("", namespace))
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete report: %s/%s", rep.Namespace, rep.GetName()))
			}
		}
	}
	return RepList, nil
}

func (s *reportStore) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	klog.Infof("watching reports rv=%s", options.ResourceVersion)
	switch options.ResourceVersion {
	case "", "0":
		return s.broadcaster.Watch()
	default:
		break
	}
	items, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}
	list, ok := items.(*openreportsv1alpha1.ReportList)
	if !ok {
		return nil, fmt.Errorf("failed to convert runtime object into report list")
	}
	events := make([]watch.Event, len(list.Items))
	for i, pol := range list.Items {
		report := pol.DeepCopy()
		events[i] = watch.Event{
			Type:   watch.Added,
			Object: report,
		}
	}
	return s.broadcaster.WatchWithPrefix(events)
}

func (s *reportStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *openreportsv1alpha1.Report:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addOpenreportsReportToTable(&table, *t)
	case *openreportsv1alpha1.ReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addOpenreportsReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (s *reportStore) NamespaceScoped() bool {
	return true
}

func (s *reportStore) GetSingularName() string {
	return "report"
}

func (s *reportStore) ShortNames() []string {
	return []string{"reps"}
}

func (s *reportStore) getRep(name, namespace string) (*openreportsv1alpha1.Report, error) {
	val, err := s.store.Reports().Get(context.TODO(), name, namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find report in store")
	}

	return val, nil
}

func (s *reportStore) listRep(namespace string) (*openreportsv1alpha1.ReportList, error) {
	valList, err := s.store.Reports().List(context.TODO(), namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find report in store")
	}

	reportList := &openreportsv1alpha1.ReportList{
		Items: make([]openreportsv1alpha1.Report, 0, len(valList)),
	}

	for _, v := range valList {
		reportList.Items = append(reportList.Items, *v.DeepCopy())
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (s *reportStore) createRep(report *openreportsv1alpha1.Report) (*openreportsv1alpha1.Report, error) {
	report.ResourceVersion = s.store.UseResourceVersion()
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return report, s.store.Reports().Create(context.TODO(), report)
}

func (s *reportStore) updateRep(report *openreportsv1alpha1.Report, _ *openreportsv1alpha1.Report) (*openreportsv1alpha1.Report, error) {
	report.ResourceVersion = s.store.UseResourceVersion()
	return report, s.store.Reports().Update(context.TODO(), report)
}

func (s *reportStore) deleteRep(report *openreportsv1alpha1.Report) error {
	return s.store.Reports().Delete(context.TODO(), report.Name, report.Namespace)
}
