package rest

import (
	"context"
	"time"

	"github.com/kyverno/reports-server/pkg/v2/logging"
	"github.com/kyverno/reports-server/pkg/v2/metrics"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

// Watch returns a watch.Interface that streams changes to the resource
// Implements rest.Watcher
func (h *GenericRESTHandler[T]) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	resourceType := h.metadata.Kind
	startTime := time.Now()

	klog.V(logging.LevelDebug).InfoS("Starting watch",
		"kind", h.metadata.Kind,
		"resourceVersion", options.ResourceVersion,
	)

	// Record watch connection
	h.metrics.Watch().RecordConnection(resourceType)
	h.metrics.Watch().IncrementActiveConnections(resourceType)

	// Create wrapped watch to track metrics on close
	var watchInterface watch.Interface
	var err error

	// For watches starting from "0" or "", return all current objects then watch for changes
	switch options.ResourceVersion {
	case "", "0":
		watchInterface, err = h.broadcaster.Watch()
	default:
		// For watches with a specific resource version, list current state first
		
		// List current objects
		items, listErr := h.List(ctx, options)
		if listErr != nil {
			return nil, listErr
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

		// Record initial events
		for range events {
			h.metrics.Watch().RecordEventSent(resourceType)
		}

		// Watch with initial events (bookmarked watch)
		watchInterface, err = h.broadcaster.WatchWithPrefix(events)
	}

	if err != nil {
		h.metrics.Watch().DecrementActiveConnections(resourceType)
		return nil, err
	}

	// Wrap watch to track when it closes
	return &metricsWatch{
		Interface:    watchInterface,
		resourceType: resourceType,
		metrics:      h.metrics,
		startTime:    startTime,
	}, nil
}

// metricsWatch wraps a watch.Interface to track metrics
type metricsWatch struct {
	watch.Interface
	resourceType string
	metrics      *metrics.Registry
	startTime    time.Time
}

// Stop wraps the underlying Stop() to record metrics
func (m *metricsWatch) Stop() {
	m.Interface.Stop()
	m.metrics.Watch().DecrementActiveConnections(m.resourceType)
	m.metrics.Watch().ObserveConnectionDuration(m.resourceType, time.Since(m.startTime).Seconds())
	
	klog.V(logging.LevelDebug).InfoS("Watch connection closed",
		"resourceType", m.resourceType,
		"duration", time.Since(m.startTime),
	)
}
