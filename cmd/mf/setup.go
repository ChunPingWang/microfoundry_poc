package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/auth"
	"github.com/younjinjeong/microfoundry/pkg/config"
	"github.com/younjinjeong/microfoundry/pkg/k8s"
)

func setupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup platform components",
	}
	cmd.AddCommand(setupKeycloakCmd())
	return cmd
}

func setupKeycloakCmd() *cobra.Command {
	var adminUser string
	var adminPass string
	var clientSecret string
	var adminPort int

	cmd := &cobra.Command{
		Use:   "keycloak",
		Short: "Deploy and configure Keycloak for authentication",
		Long:  `Deploys Keycloak to the active Kubernetes cluster and configures a realm, client, and roles for MicroFoundry authentication.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			_, clusterCfg, ok := cfg.Kubernetes.GetActiveCluster()
			if !ok {
				return fmt.Errorf("no active cluster configured")
			}

			client, err := k8s.NewClient(clusterCfg.Context, clusterCfg.Namespace, clusterCfg.Domain)
			if err != nil {
				return fmt.Errorf("connecting to cluster: %w", err)
			}

			ctx := context.Background()

			// Generate client secret if not provided
			if clientSecret == "" {
				b := make([]byte, 16)
				rand.Read(b)
				clientSecret = hex.EncodeToString(b)
			}

			// Phase 1: Deploy Keycloak
			fmt.Println("Deploying Keycloak to cluster...")
			if err := client.DeployKeycloak(ctx, adminUser, adminPass); err != nil {
				return fmt.Errorf("deploying keycloak: %w", err)
			}
			fmt.Println("  Keycloak deployment created")

			// Phase 2: Wait for readiness
			fmt.Println("Waiting for Keycloak to be ready...")
			deadline := time.After(5 * time.Minute)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

		waitLoop:
			for {
				select {
				case <-deadline:
					return fmt.Errorf("keycloak did not become ready within 5 minutes")
				case <-ticker.C:
					ready, _ := client.IsKeycloakReady(ctx)
					if ready {
						fmt.Println("  Keycloak is ready")
						break waitLoop
					}
					fmt.Print(".")
				}
			}

			// Phase 3: Port-forward and configure
			keycloakURL := fmt.Sprintf("http://localhost:%d", adminPort)
			fmt.Printf("\nTo configure Keycloak, set up port-forwarding:\n")
			fmt.Printf("  kubectl port-forward -n %s svc/keycloak %d:%d\n\n", clusterCfg.Namespace, adminPort, client.KeycloakPort())
			fmt.Println("Then run the realm configuration:")
			fmt.Printf("  mf setup keycloak-realm --url %s\n\n", keycloakURL)

			// Print configuration for mf.yaml
			fmt.Println("Add the following to your mf.yaml:")
			fmt.Println("  auth:")
			fmt.Println("    enabled: true")
			fmt.Printf("    issuer_url: \"%s/realms/microfoundry\"\n", keycloakURL)
			fmt.Println("    client_id: \"mf-admin\"")
			fmt.Printf("    client_secret: \"%s\"\n", clientSecret)
			fmt.Printf("    redirect_url: \"http://localhost:8080/auth/callback\"\n")

			fmt.Printf("\nKeycloak Admin Console: %s/admin\n", keycloakURL)
			fmt.Printf("Admin credentials: %s / %s\n", adminUser, adminPass)

			return nil
		},
	}

	cmd.Flags().StringVar(&adminUser, "admin-user", "admin", "Keycloak admin username")
	cmd.Flags().StringVar(&adminPass, "admin-pass", "admin", "Keycloak admin password")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret (auto-generated if empty)")
	cmd.Flags().IntVar(&adminPort, "port", 8180, "Local port for Keycloak access")

	return cmd
}

func setupKeycloakRealmCmd() *cobra.Command {
	var keycloakURL string
	var adminUser string
	var adminPass string
	var clientSecret string
	var redirectURI string

	cmd := &cobra.Command{
		Use:   "keycloak-realm",
		Short: "Configure Keycloak realm and client for MicroFoundry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clientSecret == "" {
				b := make([]byte, 16)
				rand.Read(b)
				clientSecret = hex.EncodeToString(b)
			}

			configurator := auth.NewKeycloakConfigurator(keycloakURL)

			fmt.Println("Configuring Keycloak realm...")
			if err := configurator.ConfigureRealm(adminUser, adminPass, clientSecret, redirectURI); err != nil {
				return fmt.Errorf("configuring realm: %w", err)
			}
			fmt.Println("  Realm 'microfoundry' configured")
			fmt.Println("  Client 'mf-admin' created")
			fmt.Println("  Roles created: platform-admin, org-admin, org-member, viewer")

			fmt.Printf("\nClient secret: %s\n", clientSecret)
			fmt.Println("\nTo add identity providers, use:")
			fmt.Println("  mf setup keycloak-idp --provider google --client-id <ID> --client-secret <SECRET>")

			return nil
		},
	}

	cmd.Flags().StringVar(&keycloakURL, "url", "http://localhost:8180", "Keycloak base URL")
	cmd.Flags().StringVar(&adminUser, "admin-user", "admin", "Keycloak admin username")
	cmd.Flags().StringVar(&adminPass, "admin-pass", "admin", "Keycloak admin password")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret (auto-generated if empty)")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "http://localhost:8080/auth/callback", "OAuth2 redirect URI")

	return cmd
}

func setupKeycloakIdPCmd() *cobra.Command {
	var keycloakURL string
	var adminUser string
	var adminPass string
	var provider string
	var clientID string
	var clientSecret string

	cmd := &cobra.Command{
		Use:   "keycloak-idp",
		Short: "Add an identity provider to Keycloak (google, github, amazon)",
		RunE: func(cmd *cobra.Command, args []string) error {
			validProviders := map[string]bool{"google": true, "github": true, "amazon": true}
			if !validProviders[provider] {
				return fmt.Errorf("provider must be one of: google, github, amazon")
			}

			configurator := auth.NewKeycloakConfigurator(keycloakURL)

			fmt.Printf("Adding %s identity provider...\n", provider)
			if err := configurator.AddIdentityProvider(adminUser, adminPass, provider, clientID, clientSecret); err != nil {
				return fmt.Errorf("adding identity provider: %w", err)
			}
			fmt.Printf("  %s identity provider configured\n", provider)

			return nil
		},
	}

	cmd.Flags().StringVar(&keycloakURL, "url", "http://localhost:8180", "Keycloak base URL")
	cmd.Flags().StringVar(&adminUser, "admin-user", "admin", "Keycloak admin username")
	cmd.Flags().StringVar(&adminPass, "admin-pass", "admin", "Keycloak admin password")
	cmd.Flags().StringVar(&provider, "provider", "", "Identity provider (google, github, amazon)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID from the provider")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret from the provider")
	_ = cmd.MarkFlagRequired("provider")
	_ = cmd.MarkFlagRequired("client-id")
	_ = cmd.MarkFlagRequired("client-secret")

	return cmd
}
