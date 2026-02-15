package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/admin"
	"github.com/younjinjeong/microfoundry/pkg/config"
	"github.com/younjinjeong/microfoundry/pkg/k8s"
)

func adminCmd() *cobra.Command {
	var port int
	var host string

	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Start the MicroFoundry admin web interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			k8sClient, err := k8s.NewClient(
				cfg.Kubernetes.Context,
				cfg.Kubernetes.Namespace,
				"cf-local.dev",
			)
			if err != nil {
				return fmt.Errorf("connecting to kubernetes: %w", err)
			}

			srv := admin.NewServer(k8sClient, cfg, version)
			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("MicroFoundry Admin starting at http://localhost:%d\n", port)
			return srv.ListenAndServe(addr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind to")

	return cmd
}
