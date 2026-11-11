package app

import (
	"fmt"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/kyverno/reports-server/pkg/app/opts"
	"k8s.io/klog/v2"
)

// setupMemLimit sets GOMEMLIMIT to a fraction of the memory limit of the
// container (cgroup) or, when there is none, of the system. The Go runtime
// treats GOMEMLIMIT as a soft limit and it does not cover non-heap memory, so
// setting it to the full cgroup limit leaves no headroom and the container is
// OOM-killed instead of the GC running harder. An explicit GOMEMLIMIT
// environment variable always wins: memlimit.Set skips itself when it is set.
func setupMemLimit(o *opts.Options) error {
	if !o.AutoMemoryLimitEnabled {
		return nil
	}
	limit, err := memlimit.Set(
		memlimit.WithRatio(o.AutoMemoryLimitRatio),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to configure GOMEMLIMIT: %w", err)
	}
	klog.Infof("GOMEMLIMIT configured to %d bytes (ratio %.2f of the detected memory limit)", limit, o.AutoMemoryLimitRatio)
	return nil
}
