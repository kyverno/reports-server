package rest

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

// Watch returns a watch.Interface that streams changes to the resource
// Implements rest.Watcher
func (h *GenericRESTHandler[T]) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	klog.V(4).InfoS("Starting watch",
		"kind", h.metadata.Kind,
		"resourceVersion", options.ResourceVersion,
	)

	// For watches starting from "0" or "", return all current objects then watch for changes
	switch options.ResourceVersion {
	case "", "0":
		return h.broadcaster.Watch()
	default:
		// For watches with a specific resource version, list current state first
		break
	}

	// List current objects
	items, err := h.List(ctx, options)
	if err != nil {
		return nil, err
	}

	// Extract items from list
	listItems := h.metadata.ListItemsFunc(items)

	// Convert to watch events
	events := make([]watch.Event, len(listItems))
	for i, item := range listItems {
		events[i] = watch.Event{
			Type:   watch.Added,
			Object: item,
		}
	}

	// Watch with initial events (bookmarked watch)
	return h.broadcaster.WatchWithPrefix(events)
}

