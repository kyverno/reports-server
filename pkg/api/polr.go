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
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type polrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Interface
}

func PolicyReportStore(store storage.Interface) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &polrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (p *polrStore) New() runtime.Object {
	return &v1alpha2.PolicyReport{}
}

func (p *polrStore) Destroy() {
}

func (p *polrStore) Kind() string {
	return "PolicyReport"
}

func (p *polrStore) NewList() runtime.Object {
	return &v1alpha2.PolicyReportList{}
}

func (p *polrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
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

	klog.Infof("listing policy reports for namespace=%s", namespace)
	list, err := p.listPolr(namespace)
	if err != nil {
		return nil, errors.NewBadRequest("failed to list resource policyreport")
	}

	// if labelSelector == labels.Everything() {
	// 	return list, nil
	// }

	polrList := &v1alpha2.PolicyReportList{
		Items: make([]v1alpha2.PolicyReport, 0),
	}
	for _, polr := range list.Items {
		if polr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(polr.Labels)) {
			polrList.Items = append(polrList.Items, polr)
		}
	}

	return polrList, nil
}

func (p *polrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)

	klog.Infof("getting policy reports name=%s namespace=%s", name, namespace)
	report, err := p.getPolr(name, namespace)
	if err != nil || report == nil {
		return nil, errors.NewNotFound(v1alpha2.Resource("policyreports"), name)
	}
	return report, nil
}

func (p *polrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
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

	polr, ok := obj.(*v1alpha2.PolicyReport)
	if !ok {
		return nil, errors.NewBadRequest("failed to validate policy report")
	}

	namespace := genericapirequest.NamespaceValue(ctx)

	if len(polr.Namespace) == 0 {
		polr.Namespace = namespace
	}

	klog.Infof("creating policy reports name=%s namespace=%s", polr.Name, polr.Namespace)
	if !isDryRun {
		err := p.createPolr(polr)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot create policy report: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Added, obj); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return obj, nil
}

func (p *polrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	oldObj, err := p.getPolr(name, namespace)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil && !forceAllowCreate {
		return nil, false, err
	}
	polr := updatedObject.(*v1alpha2.PolicyReport)

	if forceAllowCreate {
		if err := p.updatePolr(polr, oldObj); err != nil {
			klog.ErrorS(err, "failed to update resource")
		}
		if err := p.broadcaster.Action(watch.Added, updatedObject); err != nil {
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

	polr, ok := updatedObject.(*v1alpha2.PolicyReport)
	if !ok {
		return nil, false, errors.NewBadRequest("failed to validate policy report")
	}

	if len(polr.Namespace) == 0 {
		polr.Namespace = namespace
	}

	klog.Infof("updating policy reports name=%s namespace=%s", polr.Name, polr.Namespace)
	if !isDryRun {
		err := p.updatePolr(polr, oldObj)
		if err != nil {
			return nil, false, errors.NewBadRequest(fmt.Sprintf("cannot create policy report: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Modified, updatedObject); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return updatedObject, true, nil
}

func (p *polrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	polr, err := p.getPolr(name, namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to find polrs", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewNotFound(v1alpha2.Resource("policyreports"), name)
	}

	err = deleteValidation(ctx, polr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name, "namespace", klog.KRef("", namespace))
		return nil, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	klog.Infof("deleting policy reports name=%s namespace=%s", polr.Name, polr.Namespace)
	if !isDryRun {
		err = p.deletePolr(polr)
		if err != nil {
			klog.ErrorS(err, "failed to delete polr", "name", name, "namespace", klog.KRef("", namespace))
			return nil, false, errors.NewBadRequest(fmt.Sprintf("failed to delete policyreport: %s", err.Error()))
		}
		if err := p.broadcaster.Action(watch.Deleted, polr); err != nil {
			klog.ErrorS(err, "failed to broadcast event")
		}
	}

	return polr, true, nil // TODO: Add protobuf in wgpolicygroup
}

func (p *polrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	obj, err := p.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find polrs", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to find policy reports")
	}

	polrList, ok := obj.(*v1alpha2.PolicyReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse polrs", "namespace", klog.KRef("", namespace))
		return nil, errors.NewBadRequest("Failed to parse policy reports")
	}

	if !isDryRun {
		for _, polr := range polrList.Items {
			obj, isDeleted, err := p.Delete(ctx, polr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete polr", "name", polr.GetName(), "namespace", klog.KRef("", namespace))
				return nil, errors.NewBadRequest(fmt.Sprintf("Failed to delete policy report: %s/%s", polr.Namespace, polr.GetName()))
			}
			if err := p.broadcaster.Action(watch.Deleted, obj); err != nil {
				klog.ErrorS(err, "failed to broadcast event")
			}
		}
	}
	return polrList, nil
}

func (p *polrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return p.broadcaster.Watch()
}

func (p *polrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1alpha2.PolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addPolicyReportToTable(&table, *t)
	case *v1alpha2.PolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (p *polrStore) NamespaceScoped() bool {
	return true
}

func (p *polrStore) GetSingularName() string {
	return "policyreport"
}

func (p *polrStore) ShortNames() []string {
	return []string{"polr"}
}

func (p *polrStore) getPolr(name, namespace string) (*v1alpha2.PolicyReport, error) {
	val, err := p.store.PolicyReports().Get(context.TODO(), name, namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find policy report in store")
	}

	return val.DeepCopy(), nil
}

func (p *polrStore) listPolr(namespace string) (*v1alpha2.PolicyReportList, error) {
	valList, err := p.store.PolicyReports().List(context.TODO(), namespace)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find policy report in store")
	}

	reportList := &v1alpha2.PolicyReportList{
		Items: valList,
	}

	klog.Infof("value found of length:%d", len(reportList.Items))
	return reportList, nil
}

func (p *polrStore) createPolr(report *v1alpha2.PolicyReport) error {
	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	return p.store.PolicyReports().Create(context.TODO(), *report)
}

func (p *polrStore) updatePolr(report *v1alpha2.PolicyReport, oldReport *v1alpha2.PolicyReport) error {
	oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
	if err != nil {
		return errorpkg.Wrapf(err, "could not parse resource version")
	}
	report.ResourceVersion = fmt.Sprint(oldRV + 1)

	return p.store.PolicyReports().Update(context.TODO(), *report)
}

func (p *polrStore) deletePolr(report *v1alpha2.PolicyReport) error {
	return p.store.PolicyReports().Delete(context.TODO(), report.Name, report.Namespace)
}
