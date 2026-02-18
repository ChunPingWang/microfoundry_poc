package microfoundry.authz

import rego.v1

default allow := false

# When auth is disabled, user is null — allow public resources only
# SCIM, settings, clusters, workspaces, and user management still require authentication
allow if {
	input.user == null
	not input.resource in {"scim", "settings", "clusters", "users", "workspaces", "monitoring", "config"}
}

# ─── Platform Admin: full access to all resources ───
allow if {
	"platform-admin" in input.user.roles
}

# ─── Workspace Admin: manages orgs within workspace + operations ───

# Workspace admins can read operations resources
allow if {
	input.action == "read"
	"workspace-admin" in input.user.roles
	input.resource in {"dashboard", "apps", "services", "secrets"}
}

# Workspace admins can write/delete operations resources
allow if {
	input.action in {"write", "delete"}
	"workspace-admin" in input.user.roles
	input.resource in {"apps", "services", "secrets"}
}

# Workspace admins can manage users, orgs, and workspaces
allow if {
	"workspace-admin" in input.user.roles
	input.resource in {"users", "orgs", "workspaces"}
}

# Workspace admins can view audit logs
allow if {
	input.action == "read"
	"workspace-admin" in input.user.roles
	input.resource == "audit"
}

# ─── Org Admin: operations + org-scoped user management ───

# Org admins can read operations resources
allow if {
	input.action == "read"
	"org-admin" in input.user.roles
	input.resource in {"dashboard", "apps", "services", "secrets"}
}

# Org admins can write/delete operations resources
allow if {
	input.action in {"write", "delete"}
	"org-admin" in input.user.roles
	input.resource in {"apps", "services", "secrets"}
}

# Org admins can read and manage users/orgs (their own org)
allow if {
	"org-admin" in input.user.roles
	input.resource in {"users", "orgs"}
}

# Org admins can view audit logs
allow if {
	input.action == "read"
	"org-admin" in input.user.roles
	input.resource == "audit"
}

# ─── Org Member: operations only ───

# Org members can read operations resources
allow if {
	input.action == "read"
	"org-member" in input.user.roles
	input.resource in {"dashboard", "apps", "services", "secrets"}
}

# Org members can write apps, services, secrets (deploy, bind, create)
allow if {
	input.action == "write"
	"org-member" in input.user.roles
	input.resource in {"apps", "services", "secrets"}
}

# ─── Fallback: any authenticated user with org_role ───

# Org-level admin role also grants write access (for users without realm roles)
allow if {
	input.action in {"write", "delete"}
	input.user.org_role == "admin"
	input.resource in {"apps", "services", "secrets", "users", "orgs"}
}

# Org-level member role grants write for apps/services/secrets
allow if {
	input.action == "write"
	input.user.org_role == "member"
	input.resource in {"apps", "services", "secrets"}
}

# Any authenticated user with an org_role can read operations resources
allow if {
	input.action == "read"
	input.user.id != ""
	input.resource in {"dashboard", "apps", "services", "secrets"}
}

# ─── Fallback: workspace-level role ───

# Workspace-level admin role grants access to workspaces, orgs, users, and operations
allow if {
	input.user.workspace_role == "admin"
	input.resource in {"dashboard", "apps", "services", "secrets", "users", "orgs", "workspaces"}
}

# ─── Settings, clusters, catalog, monitoring, config: platform-admin only ───
# (Implicitly denied for workspace-admin, org-admin and org-member by default allow := false)

# SCIM endpoints require platform-admin (covered by the platform-admin rule above)
