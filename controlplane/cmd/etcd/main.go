package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"
	_ "k8s.io/component-base/logs/json/register"          // for JSON log format registration
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugins
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration

	"open-cluster-management.io/ocm-controlplane/pkg/etcd"
)

func main() {
	cmd := NewEtcdServerCommand()
	code := cli.Run(cmd)
	os.Exit(code)
}

// NewEtcdServerCommand creates a *cobra.Command object with default parameters
func NewEtcdServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ocm-etcd",

		// stop printing usage when the command errors
		SilenceUsage: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			// silence client-go warnings.
			// kube-apiserver loopback clients should not log self-issued warnings.
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			shutdownCtx, cancel := context.WithCancel(context.TODO())
			defer cancel()
			es := &etcd.Server{
				Dir: ".ocmconfig",
			}
			return es.Run(shutdownCtx, "2380", "2379", 0)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}
	return cmd
}
