package rest

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TableConverterFunc is a function that knows how to convert specific resource types to table rows
type TableConverterFunc func(table *metav1beta1.Table, objects ...runtime.Object)

// ConvertToTable converts a resource or list to table format for kubectl output
// Implements rest.TableConvertor
func (h *GenericRESTHandler[T]) ConvertToTable(
	ctx context.Context,
	object runtime.Object,
	tableOptions runtime.Object,
) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	// Handle single resource vs list
	switch t := object.(type) {
	case T:
		// Single resource
		table.ResourceVersion = t.GetResourceVersion()
		if h.metadata.TableConverter != nil {
			h.metadata.TableConverter(&table, t)
		} else {
			// Default: Add basic row with name
			addDefaultTableRow(&table, t)
		}

	case metav1.ListMetaAccessor:
		// List of resources
		table.ResourceVersion = t.GetListMeta().GetResourceVersion()
		table.Continue = t.GetListMeta().GetContinue()

		items := h.metadata.ListItemsFunc(object)
		if h.metadata.TableConverter != nil {
			h.metadata.TableConverter(&table, items...)
		} else {
			// Default: Add rows for all items
			for _, item := range items {
				if obj, ok := item.(metav1.Object); ok {
					addDefaultTableRow(&table, obj)
				}
			}
		}

	default:
		// Unknown type - try to handle as generic object
		if h.metadata.TableConverter != nil {
			h.metadata.TableConverter(&table, object)
		}
	}

	return &table, nil
}

// addDefaultTableRow adds a basic table row with name, namespace, and age
func addDefaultTableRow(table *metav1beta1.Table, obj metav1.Object) {
	// Define columns if not already defined
	if len(table.ColumnDefinitions) == 0 {
		table.ColumnDefinitions = []metav1beta1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Namespace", Type: "string", Format: ""},
			{Name: "Age", Type: "string", Format: ""},
		}
	}

	// Add row
	row := metav1beta1.TableRow{
		Cells: []interface{}{
			obj.GetName(),
			obj.GetNamespace(),
			translateTimestampSince(obj.GetCreationTimestamp()),
		},
		Object: runtime.RawExtension{Object: obj.(runtime.Object)},
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

