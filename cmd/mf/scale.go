package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// parseMemoryString parses CF-style memory strings like "256M", "1G", "512m", "2g" into MB.
func parseMemoryString(s string) (int, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty memory string")
	}
	if strings.HasSuffix(s, "G") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "G"))
		if err != nil {
			return 0, fmt.Errorf("invalid memory value: %s", s)
		}
		return n * 1024, nil
	}
	if strings.HasSuffix(s, "M") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "M"))
		if err != nil {
			return 0, fmt.Errorf("invalid memory value: %s", s)
		}
		return n, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value: %s (use e.g. 256M or 1G)", s)
	}
	return n, nil
}

func scaleCmd() *cobra.Command {
	var (
		instances int
		memory    string
		disk      string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "scale [app-name]",
		Short: "Scale application instances, memory, and disk",
		Long: `Scale application resources. Examples:
  mf scale hello-world -i 3
  mf scale hello-world -m 512M
  mf scale hello-world -k 1G
  mf scale hello-world -m 512M -k 1G -f`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			name := args[0]

			hasInstances := cmd.Flags().Changed("instances")
			hasMemory := memory != ""
			hasDisk := disk != ""

			if !hasInstances && !hasMemory && !hasDisk {
				return fmt.Errorf("at least one of --instances (-i), --memory (-m), or --disk (-k) is required")
			}

			k8sClient, err := newK8sClient()
			if err != nil {
				return err
			}

			// Scale instances (no restart required)
			if hasInstances {
				if instances <= 0 {
					return fmt.Errorf("--instances (-i) must be a positive number")
				}
				fmt.Printf("Scaling %s to %d instances...\n", name, instances)
				if err := k8sClient.ScaleApp(ctx, name, instances); err != nil {
					return err
				}
				fmt.Printf("App %s scaled to %d instances.\n", name, instances)
			}

			// Update memory/disk (triggers restart)
			if hasMemory || hasDisk {
				var memMB, diskMB int

				if hasMemory {
					memMB, err = parseMemoryString(memory)
					if err != nil {
						return err
					}
					if memMB < 64 || memMB > 8192 {
						return fmt.Errorf("memory must be between 64M and 8G")
					}
				}
				if hasDisk {
					diskMB, err = parseMemoryString(disk)
					if err != nil {
						return err
					}
					if diskMB < 256 || diskMB > 8192 {
						return fmt.Errorf("disk must be between 256M and 8G")
					}
				}

				// Prompt for restart unless --force
				if !force {
					fmt.Print("This will restart the app. Continue? [y/N]: ")
					scanner := bufio.NewScanner(os.Stdin)
					scanner.Scan()
					answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
					if answer != "y" && answer != "yes" {
						fmt.Println("Aborted.")
						return nil
					}
				}

				desc := []string{}
				if memMB > 0 {
					desc = append(desc, fmt.Sprintf("memory=%dM", memMB))
				}
				if diskMB > 0 {
					desc = append(desc, fmt.Sprintf("disk=%dM", diskMB))
				}
				fmt.Printf("Updating %s resources (%s)...\n", name, strings.Join(desc, ", "))

				if err := k8sClient.UpdateAppResources(ctx, name, memMB, diskMB); err != nil {
					return err
				}
				fmt.Printf("App %s resources updated. App is restarting.\n", name)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&instances, "instances", "i", 0, "Number of instances")
	cmd.Flags().StringVarP(&memory, "memory", "m", "", "Memory limit (e.g. 256M, 1G)")
	cmd.Flags().StringVarP(&disk, "disk", "k", "", "Disk limit (e.g. 256M, 1G)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force restart without prompt")

	return cmd
}
