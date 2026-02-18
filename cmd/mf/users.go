package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/auth"
	"github.com/younjinjeong/microfoundry/pkg/config"
)

// newKeycloakAdmin creates a KeycloakAdminClient from mf.yaml config.
func newKeycloakAdmin() (*auth.KeycloakAdminClient, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.Auth.AdminBaseURL == "" {
		return nil, fmt.Errorf("auth.admin_base_url is not configured in mf.yaml")
	}
	return auth.NewKeycloakAdminClient(
		cfg.Auth.AdminBaseURL,
		cfg.Auth.Realm,
		cfg.Auth.AdminClientID,
		cfg.Auth.AdminClientSecret,
	), nil
}

func usersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage Keycloak users",
		Long:  "List, create, delete, and manage users in the Keycloak identity provider.",
	}

	cmd.AddCommand(
		usersListCmd(),
		usersCreateCmd(),
		usersDeleteCmd(),
		usersToggleCmd(),
		usersResetPasswordCmd(),
		usersRolesCmd(),
		usersAssignRoleCmd(),
		usersRemoveRoleCmd(),
		usersRealmRolesCmd(),
	)

	return cmd
}

func usersListCmd() *cobra.Command {
	var search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()
			users, total, err := kc.ListUsers(ctx, search, 0, 100)
			if err != nil {
				return fmt.Errorf("listing users: %w", err)
			}

			if len(users) == 0 {
				fmt.Println("No users found.")
				return nil
			}

			fmt.Printf("%-36s %-20s %-30s %-8s %-20s\n", "ID", "USERNAME", "EMAIL", "ENABLED", "CREATED")
			for _, u := range users {
				created := ""
				if u.CreatedTimestamp > 0 {
					created = time.Unix(u.CreatedTimestamp/1000, 0).Format("2006-01-02 15:04")
				}
				enabled := "yes"
				if !u.Enabled {
					enabled = "no"
				}
				fmt.Printf("%-36s %-20s %-30s %-8s %-20s\n",
					u.ID, u.Username, u.Email, enabled, created)
			}
			fmt.Printf("\nTotal: %d users\n", total)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Filter users by username or email")
	return cmd
}

func usersCreateCmd() *cobra.Command {
	var username, email, firstName, lastName, password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || email == "" {
				return fmt.Errorf("--username and --email are required")
			}

			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			// Default first/last name to username if not provided (Keycloak requires non-empty profile)
			if firstName == "" {
				firstName = username
			}
			if lastName == "" {
				lastName = "-"
			}

			ctx := context.Background()
			user := &auth.KeycloakUser{
				Username:      username,
				Email:         email,
				FirstName:     firstName,
				LastName:      lastName,
				Enabled:       true,
				EmailVerified: true,
			}

			userID, err := kc.CreateUser(ctx, user)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}

			if password != "" {
				if err := kc.ResetPassword(ctx, userID, password, false); err != nil {
					return fmt.Errorf("user created (ID: %s) but password set failed: %w", userID, err)
				}
			}

			fmt.Printf("User created: %s (ID: %s)\n", username, userID)
			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email address (required)")
	cmd.Flags().StringVar(&firstName, "first-name", "", "First name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Last name")
	cmd.Flags().StringVar(&password, "password", "", "Initial password")
	return cmd
}

func usersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [user-id]",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := kc.DeleteUser(ctx, args[0]); err != nil {
				return fmt.Errorf("deleting user: %w", err)
			}

			fmt.Printf("User %s deleted.\n", args[0])
			return nil
		},
	}
}

func usersToggleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle [user-id]",
		Short: "Enable or disable a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()
			user, err := kc.GetUser(ctx, args[0])
			if err != nil {
				return fmt.Errorf("getting user: %w", err)
			}

			user.Enabled = !user.Enabled
			if err := kc.UpdateUser(ctx, user); err != nil {
				return fmt.Errorf("updating user: %w", err)
			}

			status := "enabled"
			if !user.Enabled {
				status = "disabled"
			}
			fmt.Printf("User %s (%s) is now %s.\n", user.Username, user.ID, status)
			return nil
		},
	}
}

func usersResetPasswordCmd() *cobra.Command {
	var password string
	var temporary bool

	cmd := &cobra.Command{
		Use:   "reset-password [user-id]",
		Short: "Reset a user's password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				return fmt.Errorf("--password is required")
			}

			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := kc.ResetPassword(ctx, args[0], password, temporary); err != nil {
				return fmt.Errorf("resetting password: %w", err)
			}

			msg := "Password reset for user %s."
			if temporary {
				msg = "Temporary password set for user %s (will be prompted to change on next login)."
			}
			fmt.Printf(msg+"\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&password, "password", "", "New password (required)")
	cmd.Flags().BoolVar(&temporary, "temporary", false, "Require password change on next login")
	return cmd
}

func usersRolesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "roles [user-id]",
		Short: "List roles assigned to a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Get user info for display
			user, err := kc.GetUser(ctx, args[0])
			if err != nil {
				return fmt.Errorf("getting user: %w", err)
			}

			roles, err := kc.GetUserRoles(ctx, args[0])
			if err != nil {
				return fmt.Errorf("getting roles: %w", err)
			}

			if len(roles) == 0 {
				fmt.Printf("User %s (%s) has no realm roles assigned.\n", user.Username, user.ID)
				return nil
			}

			fmt.Printf("Roles for %s (%s):\n", user.Username, user.Email)
			fmt.Printf("%-36s %-30s\n", "ROLE ID", "ROLE NAME")
			for _, r := range roles {
				fmt.Printf("%-36s %-30s\n", r.ID, r.Name)
			}
			return nil
		},
	}
}

// lookupRoleByName finds a realm role by name, returning its ID.
func lookupRoleByName(kc *auth.KeycloakAdminClient, name string) (auth.KeycloakRole, error) {
	ctx := context.Background()
	roles, err := kc.GetRealmRoles(ctx)
	if err != nil {
		return auth.KeycloakRole{}, fmt.Errorf("listing realm roles: %w", err)
	}
	for _, r := range roles {
		if r.Name == name {
			return r, nil
		}
	}
	return auth.KeycloakRole{}, fmt.Errorf("role %q not found in realm", name)
}

func usersAssignRoleCmd() *cobra.Command {
	var roleID, roleName string

	cmd := &cobra.Command{
		Use:   "assign-role [user-id]",
		Short: "Assign a realm role to a user",
		Long:  "Assign a realm role to a user. Use --role-name to auto-lookup the role ID, or provide both --role-id and --role-name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if roleName == "" {
				return fmt.Errorf("--role-name is required")
			}

			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			// Auto-lookup role ID if not provided
			if roleID == "" {
				role, err := lookupRoleByName(kc, roleName)
				if err != nil {
					return err
				}
				roleID = role.ID
			}

			ctx := context.Background()
			roles := []auth.KeycloakRole{{ID: roleID, Name: roleName}}
			if err := kc.AssignUserRole(ctx, args[0], roles); err != nil {
				return fmt.Errorf("assigning role: %w", err)
			}

			fmt.Printf("Role %q assigned to user %s.\n", roleName, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&roleID, "role-id", "", "Keycloak role ID (auto-resolved if omitted)")
	cmd.Flags().StringVar(&roleName, "role-name", "", "Keycloak role name (required)")
	return cmd
}

func usersRemoveRoleCmd() *cobra.Command {
	var roleID, roleName string

	cmd := &cobra.Command{
		Use:   "remove-role [user-id]",
		Short: "Remove a realm role from a user",
		Long:  "Remove a realm role from a user. Use --role-name to auto-lookup the role ID, or provide both --role-id and --role-name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if roleName == "" {
				return fmt.Errorf("--role-name is required")
			}

			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			// Auto-lookup role ID if not provided
			if roleID == "" {
				role, err := lookupRoleByName(kc, roleName)
				if err != nil {
					return err
				}
				roleID = role.ID
			}

			ctx := context.Background()
			roles := []auth.KeycloakRole{{ID: roleID, Name: roleName}}
			if err := kc.RemoveUserRole(ctx, args[0], roles); err != nil {
				return fmt.Errorf("removing role: %w", err)
			}

			fmt.Printf("Role %q removed from user %s.\n", roleName, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&roleID, "role-id", "", "Keycloak role ID (auto-resolved if omitted)")
	cmd.Flags().StringVar(&roleName, "role-name", "", "Keycloak role name (required)")
	return cmd
}

// usersRealmRolesCmd lists all available realm roles (helper for discovering role IDs).
func usersRealmRolesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "realm-roles",
		Short: "List all available realm roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := newKeycloakAdmin()
			if err != nil {
				return err
			}

			ctx := context.Background()
			roles, err := kc.GetRealmRoles(ctx)
			if err != nil {
				return fmt.Errorf("listing realm roles: %w", err)
			}

			if len(roles) == 0 {
				fmt.Println("No realm roles found.")
				return nil
			}

			fmt.Printf("%-36s %-30s\n", "ID", "NAME")
			for _, r := range roles {
				// Skip internal Keycloak roles
				if strings.HasPrefix(r.Name, "uma_") || r.Name == "offline_access" || r.Name == "default-roles-microfoundry" {
					continue
				}
				fmt.Printf("%-36s %-30s\n", r.ID, r.Name)
			}
			return nil
		},
	}
}
