package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/auth"
)

// newWorkspaceStore creates a WorkspaceStore from the active K8s cluster.
func newWorkspaceStore() (*auth.WorkspaceStore, error) {
	k8sClient, err := newK8sClient()
	if err != nil {
		return nil, err
	}
	return auth.NewWorkspaceStore(k8sClient.Clientset, k8sClient.Namespace), nil
}

func workspacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: "Manage workspaces",
		Long:  "List, create, delete, and manage workspaces and their members.",
	}

	cmd.AddCommand(
		workspacesListCmd(),
		workspacesCreateCmd(),
		workspacesDeleteCmd(),
		workspacesMembersCmd(),
		workspacesAddMemberCmd(),
		workspacesRemoveMemberCmd(),
		workspacesSetRoleCmd(),
		workspacesOrgsCmd(),
	)

	return cmd
}

func workspacesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			workspaces, err := store.List(ctx)
			if err != nil {
				return fmt.Errorf("listing workspaces: %w", err)
			}

			if len(workspaces) == 0 {
				fmt.Println("No workspaces found.")
				fmt.Println("Use 'mf workspaces create --name <name> --owner <email>' to create one.")
				return nil
			}

			fmt.Printf("%-20s %-30s %-30s %-20s\n", "ID", "NAME", "OWNER", "CREATED")
			for _, ws := range workspaces {
				created := ws.CreatedAt
				if len(created) > 16 {
					created = created[:16]
				}
				fmt.Printf("%-20s %-30s %-30s %-20s\n",
					ws.ID, ws.Name, ws.OwnerEmail, created)
			}
			return nil
		},
	}
}

func workspacesCreateCmd() *cobra.Command {
	var name, owner string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || owner == "" {
				return fmt.Errorf("--name and --owner are required")
			}

			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			ws, err := store.Create(ctx, name, owner)
			if err != nil {
				return fmt.Errorf("creating workspace: %w", err)
			}

			fmt.Printf("Workspace created:\n")
			fmt.Printf("  ID:    %s\n", ws.ID)
			fmt.Printf("  Name:  %s\n", ws.Name)
			fmt.Printf("  Owner: %s\n", ws.OwnerEmail)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Workspace name (required)")
	cmd.Flags().StringVar(&owner, "owner", "", "Owner email address (required)")
	return cmd
}

func workspacesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [ws-id]",
		Short: "Delete a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.Delete(ctx, args[0]); err != nil {
				return fmt.Errorf("deleting workspace: %w", err)
			}

			fmt.Printf("Workspace %s deleted.\n", args[0])
			return nil
		},
	}
}

func workspacesMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members [ws-id]",
		Short: "List members of a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			members, err := store.ListMembers(ctx, args[0])
			if err != nil {
				return fmt.Errorf("listing members: %w", err)
			}

			if len(members) == 0 {
				fmt.Printf("Workspace %s has no members.\n", args[0])
				return nil
			}

			fmt.Printf("Members of workspace %s:\n\n", args[0])
			fmt.Printf("%-30s %-30s %-10s %-20s\n", "EMAIL", "NAME", "ROLE", "JOINED")
			for _, m := range members {
				joined := m.JoinedAt
				if len(joined) > 16 {
					joined = joined[:16]
				}
				fmt.Printf("%-30s %-30s %-10s %-20s\n",
					m.Email, m.Name, m.Role, joined)
			}
			return nil
		},
	}
}

func workspacesAddMemberCmd() *cobra.Command {
	var email, name, role string

	cmd := &cobra.Command{
		Use:   "add-member [ws-id]",
		Short: "Add a member to a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			if name == "" {
				name = email
			}
			if role == "" {
				role = "member"
			}

			validRoles := map[string]bool{"admin": true, "member": true, "viewer": true}
			if !validRoles[role] {
				return fmt.Errorf("invalid role %q (must be admin, member, or viewer)", role)
			}

			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.AddMember(ctx, args[0], email, name, role); err != nil {
				return fmt.Errorf("adding member: %w", err)
			}

			fmt.Printf("Member %s added to workspace %s with role %s.\n", email, args[0], role)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	cmd.Flags().StringVar(&name, "name", "", "Member display name (defaults to email)")
	cmd.Flags().StringVar(&role, "role", "member", "Member role: admin, member, or viewer")
	return cmd
}

func workspacesRemoveMemberCmd() *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "remove-member [ws-id]",
		Short: "Remove a member from a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}

			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.RemoveMember(ctx, args[0], email); err != nil {
				return fmt.Errorf("removing member: %w", err)
			}

			fmt.Printf("Member %s removed from workspace %s.\n", email, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	return cmd
}

func workspacesSetRoleCmd() *cobra.Command {
	var email, role string

	cmd := &cobra.Command{
		Use:   "set-role [ws-id]",
		Short: "Change a member's role in a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || role == "" {
				return fmt.Errorf("--email and --role are required")
			}

			validRoles := map[string]bool{"admin": true, "member": true, "viewer": true}
			if !validRoles[role] {
				return fmt.Errorf("invalid role %q (must be admin, member, or viewer)", role)
			}

			store, err := newWorkspaceStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.SetMemberRole(ctx, args[0], email, role); err != nil {
				return fmt.Errorf("setting role: %w", err)
			}

			fmt.Printf("Member %s role set to %s in workspace %s.\n", email, role, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	cmd.Flags().StringVar(&role, "role", "", "New role: admin, member, or viewer (required)")
	return cmd
}

func workspacesOrgsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "orgs [ws-id]",
		Short: "List organizations in a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			orgs, err := store.ListByWorkspace(ctx, args[0])
			if err != nil {
				return fmt.Errorf("listing organizations: %w", err)
			}

			if len(orgs) == 0 {
				fmt.Printf("Workspace %s has no organizations.\n", args[0])
				return nil
			}

			fmt.Printf("Organizations in workspace %s:\n\n", args[0])
			fmt.Printf("%-20s %-30s %-30s %-20s\n", "ID", "NAME", "OWNER", "NAMESPACE")
			for _, o := range orgs {
				fmt.Printf("%-20s %-30s %-30s %-20s\n",
					o.ID, o.Name, o.OwnerEmail, o.Namespace)
			}
			return nil
		},
	}
}
