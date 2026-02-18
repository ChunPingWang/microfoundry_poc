package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/auth"
	"github.com/younjinjeong/microfoundry/pkg/config"
	"golang.org/x/term"
)

func loginCmd() *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with MicroFoundry",
		Long:  "Login to MicroFoundry using Keycloak credentials. Stores an access token for subsequent CLI commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if !cfg.Auth.Enabled {
				return fmt.Errorf("authentication is not enabled in mf.yaml")
			}
			if cfg.Auth.IssuerURL == "" {
				return fmt.Errorf("auth.issuer_url is not configured in mf.yaml")
			}

			// Interactive prompts if flags not provided
			if username == "" {
				fmt.Print("Username: ")
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					username = strings.TrimSpace(scanner.Text())
				}
				if username == "" {
					return fmt.Errorf("username is required")
				}
			}

			if password == "" {
				fmt.Print("Password: ")
				pwBytes, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("reading password: %w", err)
				}
				fmt.Println() // newline after hidden input
				password = string(pwBytes)
				if password == "" {
					return fmt.Errorf("password is required")
				}
			}

			token, err := auth.LoginWithPassword(
				cfg.Auth.IssuerURL,
				cfg.Auth.ClientID,
				cfg.Auth.ClientSecret,
				username,
				password,
			)
			if err != nil {
				return err
			}

			if err := auth.SaveToken(token); err != nil {
				return fmt.Errorf("saving token: %w", err)
			}

			roles := "none"
			if len(token.Roles) > 0 {
				roles = strings.Join(token.Roles, ", ")
			}
			fmt.Printf("Logged in as %s (%s)\n", token.Username, token.Email)
			fmt.Printf("Roles: %s\n", roles)
			return nil
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "Username")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Password")
	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored authentication token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.ClearToken(); err != nil {
				return fmt.Errorf("clearing token: %w", err)
			}
			fmt.Println("Logged out.")
			return nil
		},
	}
}

func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.LoadToken()
			if err != nil {
				return err
			}

			// Refresh if expired
			if token.IsExpired() {
				cfg, err := config.Load()
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				if err := auth.RefreshToken(
					cfg.Auth.IssuerURL,
					cfg.Auth.ClientID,
					cfg.Auth.ClientSecret,
					token,
				); err != nil {
					return err
				}
				if err := auth.SaveToken(token); err != nil {
					return fmt.Errorf("saving refreshed token: %w", err)
				}
			}

			roles := "none"
			if len(token.Roles) > 0 {
				roles = strings.Join(token.Roles, ", ")
			}
			fmt.Printf("Username:  %s\n", token.Username)
			fmt.Printf("Email:     %s\n", token.Email)
			fmt.Printf("Roles:     %s\n", roles)
			fmt.Printf("Expires:   %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}
