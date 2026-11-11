package api

import (
	"strconv"

	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// sendInitialEvents reports whether the request is a streaming list (watch=true
// with sendInitialEvents=true). Such a watch must start with the current state
// of the collection and then emit a bookmark that ends the initial-events phase,
// see https://kubernetes.io/docs/reference/using-api/api-concepts/#streaming-lists
func sendInitialEvents(options *metainternalversion.ListOptions) bool {
	return options != nil && options.SendInitialEvents != nil && *options.SendInitialEvents
}

// initialEventsEndBookmark builds the BOOKMARK event that terminates the
// initial events of a streaming list. bookmark must be an empty object of the
// watched kind; it gets the "k8s.io/initial-events-end" annotation and a
// resourceVersion at least as high as every initial event and as
// listResourceVersion, so a client that resumes from the bookmark neither
// repeats nor misses an event. The initial events may come in map or database
// order, so no ordering is assumed.
func initialEventsEndBookmark[T interface {
	runtime.Object
	metav1.Object
}](initialEvents []watch.Event, listResourceVersion string, bookmark T) watch.Event {
	maxRV, _ := strconv.ParseUint(listResourceVersion, 10, 64)
	for _, event := range initialEvents {
		accessor, err := meta.Accessor(event.Object)
		if err != nil {
			continue
		}
		if rv, err := strconv.ParseUint(accessor.GetResourceVersion(), 10, 64); err == nil && rv > maxRV {
			maxRV = rv
		}
	}
	bookmark.SetResourceVersion(strconv.FormatUint(maxRV, 10))
	bookmark.SetAnnotations(map[string]string{metav1.InitialEventsAnnotationKey: "true"})
	return watch.Event{Type: watch.Bookmark, Object: bookmark}
}
