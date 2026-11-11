package api

import (
	"context"
	"testing"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/inmemory"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

// receiveEvent returns the next event of w or fails the test when none arrives in time.
func receiveEvent(t *testing.T, w watch.Interface) watch.Event {
	t.Helper()
	select {
	case event, ok := <-w.ResultChan():
		if !ok {
			t.Fatalf("watch channel closed unexpectedly")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for a watch event")
	}
	return watch.Event{}
}

// watchListOptions builds the options client-go sends for a streaming list.
func watchListOptions(selector labels.Selector) *metainternalversion.ListOptions {
	sendInitialEvents := true
	return &metainternalversion.ListOptions{
		Watch:                true,
		SendInitialEvents:    &sendInitialEvents,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
		LabelSelector:        selector,
	}
}

// TestEphemeralReportWatchListEmitsInitialEventsAndBookmark checks that a
// streaming list replays the filtered current objects as ADDED events and then
// emits a BOOKMARK annotated with k8s.io/initial-events-end whose
// resourceVersion is at least as high as every replayed object, whatever order
// the store returns them in.
func TestEphemeralReportWatchListEmitsInitialEventsAndBookmark(t *testing.T) {
	store := inmemory.New().EphemeralReports()
	ctx := context.Background()
	for _, report := range []*reportsv1.EphemeralReport{
		{ObjectMeta: metav1.ObjectMeta{Name: "low", Namespace: "ns", ResourceVersion: "5", Labels: map[string]string{"app": "a"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "high", Namespace: "ns", ResourceVersion: "12", Labels: map[string]string{"app": "a"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "ns", ResourceVersion: "7", Labels: map[string]string{"app": "b"}}},
	} {
		if err := store.Create(ctx, report); err != nil {
			t.Fatalf("create %s: %v", report.Name, err)
		}
	}

	api := EphemeralReportStore(store)
	w, err := api.Watch(ctx, watchListOptions(labels.SelectorFromSet(labels.Set{"app": "a"})))
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Stop()

	added := map[string]bool{}
	for i := 0; i < 2; i++ {
		event := receiveEvent(t, w)
		if event.Type != watch.Added {
			t.Fatalf("event %d: expected %s, got %s", i, watch.Added, event.Type)
		}
		report, ok := event.Object.(*reportsv1.EphemeralReport)
		if !ok {
			t.Fatalf("event %d: unexpected object type %T", i, event.Object)
		}
		added[report.Name] = true
	}
	if !added["low"] || !added["high"] || len(added) != 2 {
		t.Fatalf("expected ADDED events for low and high only, got %v", added)
	}

	bookmark := receiveEvent(t, w)
	if bookmark.Type != watch.Bookmark {
		t.Fatalf("expected %s after the initial events, got %s", watch.Bookmark, bookmark.Type)
	}
	report, ok := bookmark.Object.(*reportsv1.EphemeralReport)
	if !ok {
		t.Fatalf("bookmark: unexpected object type %T", bookmark.Object)
	}
	if got := report.Annotations[metav1.InitialEventsAnnotationKey]; got != "true" {
		t.Fatalf("bookmark: expected annotation %s=true, got %q", metav1.InitialEventsAnnotationKey, got)
	}
	if report.ResourceVersion != "12" {
		t.Fatalf("bookmark: expected resourceVersion 12 (highest initial event), got %q", report.ResourceVersion)
	}
}

// TestClusterPolicyReportWatchListEmptyCollectionEmitsBookmark checks that an
// empty collection still ends its initial-events phase with the bookmark.
func TestClusterPolicyReportWatchListEmptyCollectionEmitsBookmark(t *testing.T) {
	api := ClusterPolicyReportStore(inmemory.New().ClusterPolicyReports())
	w, err := api.Watch(context.Background(), watchListOptions(nil))
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Stop()

	bookmark := receiveEvent(t, w)
	if bookmark.Type != watch.Bookmark {
		t.Fatalf("expected %s, got %s", watch.Bookmark, bookmark.Type)
	}
	report, ok := bookmark.Object.(*v1alpha2.ClusterPolicyReport)
	if !ok {
		t.Fatalf("bookmark: unexpected object type %T", bookmark.Object)
	}
	if got := report.Annotations[metav1.InitialEventsAnnotationKey]; got != "true" {
		t.Fatalf("bookmark: expected annotation %s=true, got %q", metav1.InitialEventsAnnotationKey, got)
	}
	if report.ResourceVersion == "" {
		t.Fatalf("bookmark: expected a resourceVersion to be set")
	}
}

// TestWatchWithoutSendInitialEventsIsUnchanged checks that a plain watch from
// resourceVersion "0" keeps its existing behaviour: no replay of the current
// objects and no bookmark.
func TestWatchWithoutSendInitialEventsIsUnchanged(t *testing.T) {
	store := inmemory.New().EphemeralReports()
	ctx := context.Background()
	if err := store.Create(ctx, &reportsv1.EphemeralReport{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "ns", ResourceVersion: "3"}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	api := EphemeralReportStore(store)
	w, err := api.Watch(ctx, &metainternalversion.ListOptions{Watch: true, ResourceVersion: "0"})
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Stop()

	select {
	case event := <-w.ResultChan():
		t.Fatalf("expected no event, got %s", event.Type)
	case <-time.After(100 * time.Millisecond):
	}
}
