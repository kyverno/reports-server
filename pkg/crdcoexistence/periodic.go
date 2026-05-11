package crdcoexistence

import (
	"context"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// PeriodicallyCheckForNewConflicts re-runs CRD conflict detection on a timer.
// If a new conflicting CRD appears that was not present at startup, the cancel
// function is called to trigger a graceful restart.
func PeriodicallyCheckForNewConflicts(ctx context.Context, cancel context.CancelFunc, restConfig *rest.Config, initialPrefixes []string) {
	known := make(map[string]bool, len(initialPrefixes))
	for _, p := range initialPrefixes {
		known[p] = true
	}

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := DetectConflictingCRDs(restConfig)
			for _, p := range current {
				if !known[p] {
					klog.InfoS("new conflicting CRD detected since startup, restarting to apply OpenAPI suppression", "prefix", p)
					cancel()
					return
				}
			}
		}
	}
}
