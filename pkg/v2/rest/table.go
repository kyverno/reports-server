package rest

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConvertToTable converts a resource or list to table format for kubectl output
// Implements rest.TableConvertor
func (h *GenericRESTHandler[T]) ConvertToTable(
	ctx context.Context,
	object runtime.Object,
	tableOptions runtime.Object,
) (*metav1beta1.Table, error) {
	table := &metav1beta1.Table{}

	// Handle single resource vs list
	switch t := object.(type) {
	case T:
		h.convertSingleObject(table, t)
	case metav1.ListMetaAccessor:
		h.convertListOfObjects(table, object, t)
	default:
		h.convertUnknownObject(table, object)
	}

	return table, nil
}

// convertSingleObject converts a single resource to table format
func (h *GenericRESTHandler[T]) convertSingleObject(table *metav1beta1.Table, obj T) {
	table.ResourceVersion = obj.GetResourceVersion()

	if h.metadata.TableConverter != nil {
		h.metadata.TableConverter(table, obj)
		return
	}

	h.addDefaultTableRow(table, obj)
}

// convertListOfObjects converts a list of resources to table format
func (h *GenericRESTHandler[T]) convertListOfObjects(
	table *metav1beta1.Table,
	listObject runtime.Object,
	listMeta metav1.ListMetaAccessor,
) {
	// Set list metadata
	table.ResourceVersion = listMeta.GetListMeta().GetResourceVersion()

	// Extract and convert items
	items := h.metadata.ListItemsFunc(listObject)
	typedItems := h.extractTypedItems(items)

	// Convert to table rows
	if h.metadata.TableConverter != nil {
		runtimeItems := h.convertToRuntimeObjects(typedItems)
		h.metadata.TableConverter(table, runtimeItems...)
		return
	}

	// Use default converter
	for _, item := range typedItems {
		h.addDefaultTableRow(table, item)
	}
}

// convertUnknownObject attempts to convert an unknown object type
func (h *GenericRESTHandler[T]) convertUnknownObject(table *metav1beta1.Table, obj runtime.Object) {
	if h.metadata.TableConverter != nil {
		h.metadata.TableConverter(table, obj)
	}
}

// extractTypedItems converts []runtime.Object to []T
func (h *GenericRESTHandler[T]) extractTypedItems(items []runtime.Object) []T {
	typedItems := make([]T, 0, len(items))
	for _, item := range items {
		if typedItem, ok := item.(T); ok {
			typedItems = append(typedItems, typedItem)
		}
	}
	return typedItems
}

// convertToRuntimeObjects converts []T to []runtime.Object
func (h *GenericRESTHandler[T]) convertToRuntimeObjects(items []T) []runtime.Object {
	runtimeItems := make([]runtime.Object, len(items))
	for i, item := range items {
		runtimeItems[i] = item
	}
	return runtimeItems
}

// getDefaultColumnDefinitions returns the default table column definitions
func getDefaultColumnDefinitions() []metav1beta1.TableColumnDefinition {
	return []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name"},
		{Name: "Namespace", Type: "string", Format: ""},
		{Name: "Age", Type: "string", Format: ""},
	}
}

// addDefaultTableRow adds a basic table row with name, namespace, and age
func (h *GenericRESTHandler[T]) addDefaultTableRow(table *metav1beta1.Table, obj T) {
	// Ensure columns are defined
	if len(table.ColumnDefinitions) == 0 {
		table.ColumnDefinitions = getDefaultColumnDefinitions()
	}

	// Add row
	row := metav1beta1.TableRow{
		Cells: []interface{}{
			obj.GetName(),
			obj.GetNamespace(),
			translateTimestampSince(obj.GetCreationTimestamp()),
		},
		Object: runtime.RawExtension{Object: obj},
	}
	table.Rows = append(table.Rows, row)
}

// translateTimestampSince returns the elapsed time since timestamp in a human-readable format
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	duration := metav1.Now().Sub(timestamp.Time)

	// Format similar to kubectl
	if duration.Seconds() < 60 {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration.Minutes() < 60 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
