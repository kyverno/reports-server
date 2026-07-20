package app

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/kyverno/reports-server/pkg/app/opts"
	"github.com/kyverno/reports-server/pkg/crdcoexistence"
	"github.com/kyverno/reports-server/pkg/server"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version"
	"k8s.io/klog/v2"
)

func NewPolicyServer(stopCh <-chan struct{}) *cobra.Command {
	opts := opts.NewOptions()
	cmd := &cobra.Command{
		Short:        "Launch reports-server",
		Long:         "Launch reports-server",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := runCommand(opts, stopCh); err != nil {
				return err
			}
			return nil
		},
	}
	fs := cmd.Flags()
	nfs := opts.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	logs.AddGoFlags(local)
	nfs.FlagSet("logging").AddGoFlagSet(local)

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, cols)
	})
	fs.AddGoFlagSet(local)
	return cmd
}

func runCommand(o *opts.Options, stopCh <-chan struct{}) error {
	if o.ShowVersion {
		fmt.Println(version.Get().GitVersion)
		os.Exit(0)
	}
	errors := o.Validate()
	if len(errors) > 0 {
		return errors[0]
	}

	podIp := os.Getenv("POD_IP")
	if podIp == "" {
		podIp, _ = os.Hostname()
	}

	headlessSvc := os.Getenv("HEADLESS_SERVICE")
	if headlessSvc == "" {
		klog.Error("no headless service dns name found. api service cleanup during leader shutdown will not work properly")
	}

	config, err := server.NewServerConfig(context.TODO(), *o)
	if err != nil {
		return err
	}
	s, err := config.Complete()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopCh
		cancel()
	}()
	go crdcoexistence.PeriodicallyCheckForNewConflicts(ctx, cancel, config.Rest, config.OpenAPIIgnorePrefixes)

	go func() {
		if err := s.RunUntil(ctx.Done()); err != nil {
			klog.ErrorS(err, "failed to run server")
			os.Exit(1)
		}
	}()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "reports-server-leader",
			Namespace: "kube-system",
		},
		Client: config.KubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: podIp,
		},
	}

	reconcilerCtx, reconcilerCancel := context.WithCancel(context.Background())
	reconcilerDone := make(chan struct{})
	var wasLeader atomic.Bool

	go leaderelection.RunOrDie(context.Background(), leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				wasLeader.Store(true)
				go func() {
					select {
					//  OnStoppedLeading ran first and cancelled the context. replace it with a new one
					case <-reconcilerCtx.Done():
						reconcilerCtx, reconcilerCancel = context.WithCancel(context.Background())
					default:
					}
					config.StartAPIServiceReconciler(reconcilerCtx)
				}()

				<-stopCh
				reconcilerCancel()

				resolver := net.Resolver{}
				addrs, err := resolver.LookupHost(ctx, headlessSvc)
				if err != nil {
					klog.Errorf("error looking up headless service dns name: %s, %s", headlessSvc, err.Error())
				}

				// check if this is the only pod remaining
				otherReplicasExist := false
				for _, a := range addrs {
					if a == podIp {
						continue
					}
					// there's an address and its not that of the current pod
					otherReplicasExist = true
				}

				// if its, clean up the apiservices (convert to local) on shutdown
				if !otherReplicasExist {
					if err := config.CleanupApiServices(); err != nil {
						klog.ErrorS(err, "failed to cleanup api-services during shutdown")
					}
				}
				close(reconcilerDone)
			},
			OnStoppedLeading: func() {
				wasLeader.Store(false)
				reconcilerCancel()
			},
			OnNewLeader: func(identity string) {
			},
		},
	})

	// wait on shutdown signal
	<-stopCh
	klog.Info("received shutdown signal")

	// wait on the cleanup to finish only if this was the leader
	if wasLeader.Load() {
		<-reconcilerDone
		klog.Info("cleanup completed")
	}

	return nil
}
