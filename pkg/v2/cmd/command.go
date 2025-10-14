package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version"
	"k8s.io/klog/v2"
)

// NewCommand creates the reports-server command
func NewCommand() *cobra.Command {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:   "reports-server",
		Short: "Kubernetes aggregated API server for policy reports",
		Long: `reports-server is a Kubernetes aggregated API server that provides
storage and querying capabilities for policy reports, ephemeral reports, and open reports.

It implements the following API groups:
  - wgpolicyk8s.io/v1alpha2 (PolicyReport, ClusterPolicyReport)
  - reports.kyverno.io/v1 (EphemeralReport, ClusterEphemeralReport)
  - openreports.io/v1alpha1 (Report, ClusterReport)

Storage backends supported:
  - PostgreSQL (production)
  - etcd (production)
  - In-memory (development/testing)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle version flag
			if opts.ShowVersion {
				fmt.Println(version.Get().GitVersion)
				return nil
			}

			// Validate options
			if errs := opts.Validate(); len(errs) > 0 {
				return errs[0]
			}

			// Run server
			return Run(opts)
		},
		SilenceUsage: true,
	}

	// Add flags
	flags := opts.Flags()
	for _, f := range flags.FlagSets {
		cmd.Flags().AddFlagSet(f)
	}

	// Setup usage and help
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), "Usage:\n  %s\n\n", cmd.UseLine())
		flag.PrintSections(cmd.OutOrStderr(), flags, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\nUsage:\n  %s\n\n", cmd.Long, cmd.UseLine())
		flag.PrintSections(cmd.OutOrStdout(), flags, cols)
	})

	return cmd
}

// Run runs the reports server with the given options
func Run(opts *Options) error {
	// Initialize logging
	logs.InitLogs()
	defer logs.FlushLogs()

	klog.InfoS("Starting reports-server",
		"version", version.Get().GitVersion,
		"storage", opts.StorageBackend,
	)

	// Create server configuration
	config, err := opts.Complete()
	if err != nil {
		return fmt.Errorf("failed to create server config: %w", err)
	}

	// Create server
	srv, err := config.Complete()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		klog.InfoS("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Run server
	klog.Info("Server starting...")
	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server exited with error: %w", err)
	}

	klog.Info("Server stopped gracefully")
	return nil
}
