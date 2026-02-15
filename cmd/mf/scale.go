package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func scaleCmd() *cobra.Command {
	var instances int

	cmd := &cobra.Command{
		Use:   "scale [app-name]",
		Short: "Scale application instances",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			name := args[0]

			if instances <= 0 {
				return fmt.Errorf("--instances (-i) must be a positive number")
			}

			k8sClient, err := newK8sClient()
			if err != nil {
				return err
			}

			fmt.Printf("Scaling %s to %d instances...\n", name, instances)
			if err := k8sClient.ScaleApp(ctx, name, instances); err != nil {
				return err
			}

			fmt.Printf("App %s scaled to %d instances.\n", name, instances)
			return nil
		},
	}

	cmd.Flags().IntVarP(&instances, "instances", "i", 0, "Number of instances")
	_ = cmd.MarkFlagRequired("instances")

	return cmd
}
