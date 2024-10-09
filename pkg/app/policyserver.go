package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/kyverno/reports-server/pkg/app/opts"
	"github.com/kyverno/reports-server/pkg/storage/etcd"
	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version"
	"k8s.io/klog/v2"
)

func NewPolicyServer(stopCh <-chan struct{}) *cobra.Command {
	opts := opts.NewOptions()
	cmd := &cobra.Command{
		Short: "Launch reports-server",
		Long:  "Launch reports-server",
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
	config, err := o.ServerConfig()
	if err != nil {
		return err
	}

	if o.Debug {
		go func() {
			klog.InfoS("starting embedded etcd etcd in directory=%s", o.EtcdDir)
			err := etcd.StartETCDServer(stopCh, o.EtcdDir)
			if err != nil {
				klog.ErrorS(err, "failed to start etcd server")
				os.Exit(1)
			}
		}()
	}

	s, err := config.Complete()
	if err != nil {
		return err
	}
	return s.RunUntil(stopCh)
}
