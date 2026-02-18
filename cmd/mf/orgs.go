package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/auth"
)

// newOrgStore creates an OrgStore from the active K8s cluster.
func newOrgStore() (*auth.OrgStore, error) {
	k8sClient, err := newK8sClient()
	if err != nil {
		return nil, err
	}
	return auth.NewOrgStore(k8sClient.Clientset, k8sClient.Namespace), nil
}

func orgsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orgs",
		Short: "Manage organizations",
		Long:  "List, create, delete, and manage organizations and their members.",
	}

	cmd.AddCommand(
		orgsListCmd(),
		orgsCreateCmd(),
		orgsDeleteCmd(),
		orgsMembersCmd(),
		orgsAddMemberCmd(),
		orgsRemoveMemberCmd(),
		orgsSetRoleCmd(),
	)

	return cmd
}

func orgsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			orgs, err := store.List(ctx)
			if err != nil {
				return fmt.Errorf("listing organizations: %w", err)
			}

			if len(orgs) == 0 {
				fmt.Println("No organizations found.")
				fmt.Println("Use 'mf orgs create --name <name> --owner <email>' to create one.")
				return nil
			}

			fmt.Printf("%-20s %-30s %-30s %-20s %-20s\n", "ID", "NAME", "OWNER", "NAMESPACE", "CREATED")
			for _, o := range orgs {
				created := o.CreatedAt
				if len(created) > 16 {
					created = created[:16]
				}
				fmt.Printf("%-20s %-30s %-30s %-20s %-20s\n",
					o.ID, o.Name, o.OwnerEmail, o.Namespace, created)
			}
			return nil
		},
	}
}

func orgsCreateCmd() *cobra.Command {
	var name, owner string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || owner == "" {
				return fmt.Errorf("--name and --owner are required")
			}

			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			org, err := store.Create(ctx, name, owner)
			if err != nil {
				return fmt.Errorf("creating organization: %w", err)
			}

			fmt.Printf("Organization created:\n")
			fmt.Printf("  ID:        %s\n", org.ID)
			fmt.Printf("  Name:      %s\n", org.Name)
			fmt.Printf("  Owner:     %s\n", org.OwnerEmail)
			fmt.Printf("  Namespace: %s\n", org.Namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Organization name (required)")
	cmd.Flags().StringVar(&owner, "owner", "", "Owner email address (required)")
	return cmd
}

func orgsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [org-id]",
		Short: "Delete an organization",
		Long:  "Delete an organization, its member list, and its Kubernetes namespace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.Delete(ctx, args[0]); err != nil {
				return fmt.Errorf("deleting organization: %w", err)
			}

			fmt.Printf("Organization %s deleted.\n", args[0])
			return nil
		},
	}
}

func orgsMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members [org-id]",
		Short: "List members of an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			members, err := store.ListMembers(ctx, args[0])
			if err != nil {
				return fmt.Errorf("listing members: %w", err)
			}

			if len(members) == 0 {
				fmt.Printf("Organization %s has no members.\n", args[0])
				return nil
			}

			fmt.Printf("Members of organization %s:\n\n", args[0])
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

func orgsAddMemberCmd() *cobra.Command {
	var email, name, role string

	cmd := &cobra.Command{
		Use:   "add-member [org-id]",
		Short: "Add a member to an organization",
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

			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.AddMember(ctx, args[0], email, name, role); err != nil {
				return fmt.Errorf("adding member: %w", err)
			}

			fmt.Printf("Member %s added to organization %s with role %s.\n", email, args[0], role)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	cmd.Flags().StringVar(&name, "name", "", "Member display name (defaults to email)")
	cmd.Flags().StringVar(&role, "role", "member", "Member role: admin, member, or viewer")
	return cmd
}

func orgsRemoveMemberCmd() *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "remove-member [org-id]",
		Short: "Remove a member from an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}

			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.RemoveMember(ctx, args[0], email); err != nil {
				return fmt.Errorf("removing member: %w", err)
			}

			fmt.Printf("Member %s removed from organization %s.\n", email, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	return cmd
}

func orgsSetRoleCmd() *cobra.Command {
	var email, role string

	cmd := &cobra.Command{
		Use:   "set-role [org-id]",
		Short: "Change a member's role in an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || role == "" {
				return fmt.Errorf("--email and --role are required")
			}

			validRoles := map[string]bool{"admin": true, "member": true, "viewer": true}
			if !validRoles[role] {
				return fmt.Errorf("invalid role %q (must be admin, member, or viewer)", role)
			}

			store, err := newOrgStore()
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := store.SetMemberRole(ctx, args[0], email, role); err != nil {
				return fmt.Errorf("setting role: %w", err)
			}

			fmt.Printf("Member %s role set to %s in organization %s.\n", email, role, args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Member email address (required)")
	cmd.Flags().StringVar(&role, "role", "", "New role: admin, member, or viewer (required)")
	return cmd
}
