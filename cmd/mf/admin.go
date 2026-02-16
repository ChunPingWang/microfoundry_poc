package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/admin"
	"github.com/younjinjeong/microfoundry/pkg/config"
	"github.com/younjinjeong/microfoundry/pkg/k8s"
	"github.com/younjinjeong/microfoundry/pkg/monitoring"
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

			clientManager := k8s.NewClientManager(
				cfg.Kubernetes.Clusters,
				cfg.Kubernetes.Active,
			)

			srv := admin.NewServer(clientManager, cfg, version)

			// Start background metrics collector
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go monitoring.StartCollector(ctx, srv.GetMetrics(), clientManager, 30*time.Second)

			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("MicroFoundry Admin starting at http://localhost:%d\n", port)
			fmt.Printf("Active cluster: %s\n", cfg.Kubernetes.Active)
			return srv.ListenAndServe(addr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host to bind to")

	return cmd
}
