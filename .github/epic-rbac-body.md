## Summary

Implement end-to-end Role-Based Access Control (RBAC) across all MicroFoundry layers: CLI authentication (`mf login`), Keycloak realm roles, OPA policies, Web UI menu visibility, and API authorization. Defines three user types with distinct permissions and menu access.

## User Types

### 1. Super Admin (`platform-admin` realm role)
- **Sees all menus**: Operations (Dashboard, Applications, Services, Secrets) + Settings (Clusters, Catalog, Registry, Webhooks, SMTP, Endpoints, Metrics, Users & Orgs, Platform)
- **Full access**: Can see all Organizations across the platform, manage all users, configure platform settings, deploy/manage all apps
- **Example user**: `myadmin`
- **Keycloak realm role**: `platform-admin`

### 2. Org Admin (`org-admin` realm role + org membership `admin`)
- **Sees**: Operations menu (Dashboard, Applications, Services, Secrets) + Users & Orgs menu (org-scoped: manage their org's users, invitations, permissions)
- **Cannot see**: Settings menu (Clusters, Catalog, Registry, Webhooks, SMTP, Endpoints, Metrics, Platform)
- **Scope**: Can only see and manage resources within their own Organization(s)
- **Keycloak realm role**: `org-admin`
- **Org membership role**: `admin`

### 3. General User / Developer (`org-member` realm role + org membership `member`)
- **Sees**: Operations menu only (Dashboard, Applications, Services, Secrets)
- **Cannot see**: Settings menu, Users & Orgs menu
- **Capabilities**: Deploy apps, bind/create services, manage secrets - scoped to their org
- **Menu access granularity**: Org Admin can further restrict which Operations sub-menus are visible per user
- **Keycloak realm role**: `org-member`
- **Org membership role**: `member` or `viewer`

## Architecture

### Layer 1: `mf login` - CLI Authentication
Currently the CLI uses direct K8s kubeconfig + Keycloak admin credentials from `mf.yaml`. All CLI commands that hit the MicroFoundry API should use an access token obtained via `mf login`.

| Command | Description |
|---------|-------------|
| `mf login` | Interactive login: opens browser for OIDC flow or prompts for username/password (Resource Owner Password grant). Stores access token + refresh token in `~/.mf/token.json` |
| `mf login --username USER --password PASS` | Non-interactive login for CI/CD |
| `mf logout` | Clear stored tokens |
| `mf whoami` | Display current authenticated user, roles, and active org |

**Token Storage**: `~/.mf/token.json`

**Token Injection**: All CLI commands that call the admin API should read the stored token and pass it as `Authorization: Bearer <token>` header. Commands that only use K8s client-go (e.g., `mf push`, `mf apps`) continue using kubeconfig but should validate the user role first.

### Layer 2: Keycloak Realm Roles

Update `keycloak_setup.go` to create the canonical roles and assign them during user creation:

| Realm Role | Purpose | Auto-assigned |
|------------|---------|---------------|
| `platform-admin` | Super admin - full platform access | Manual assignment only |
| `org-admin` | Organization administrator | When user creates an org or is set as org admin |
| `org-member` | General developer/user | Default role for new users / org invitees |

### Layer 3: OPA Policy Updates

Update `authz.rego` to enforce the 3-tier model with explicit resource categorization:

- **Operations resources**: `dashboard`, `apps`, `services`, `secrets`
- **Settings resources**: `settings`, `clusters`, `catalog`, `monitoring`, `config`
- **User management**: `users`, `orgs`, `scim`

Platform-admin: all resources. Org-admin: operations + own org user management. Org-member: operations only.

### Layer 4: Web UI - Role-Based Menu Visibility

Update `nav.html` to conditionally render menu sections based on user roles:

| Menu Section | Menu Item | platform-admin | org-admin | org-member |
|-------------|-----------|:-:|:-:|:-:|
| **Operations** | Dashboard | Y | Y | Y |
| | Applications | Y | Y | Y |
| | Services | Y | Y | Y |
| | Secrets | Y | Y | Y |
| **Settings** | Clusters | Y | - | - |
| | Service Catalog | Y | - | - |
| | Registry | Y | - | - |
| | Webhooks | Y | - | - |
| | SMTP | Y | - | - |
| | Endpoints | Y | - | - |
| | Metrics & Alerts | Y | - | - |
| | Users & Orgs | Y | Y (org-scoped) | - |
| | Platform | Y | - | - |

### Layer 5: Org-Scoped Resource Filtering

When an org-admin or org-member views resources (apps, services, secrets), they should only see resources belonging to their active organization. Platform-admin sees all resources across all orgs.

## Sub-Tasks

### Task 1: `mf login` / `mf logout` / `mf whoami` CLI commands
- Add `cmd/mf/login.go` with OIDC Resource Owner Password Credentials flow
- Token storage in `~/.mf/token.json` with auto-refresh
- `mf whoami` shows username, email, roles, active org
- Add `--username` / `--password` flags for non-interactive login
- Enable `directAccessGrantsEnabled` on mf-admin client for password grant

### Task 2: CLI Token Injection
- Create `pkg/auth/token.go` - load/refresh/save token utilities
- Update CLI commands (`users`, `orgs`, and future API-calling commands) to attach Bearer token
- Add `mf login` requirement check before API-calling commands

### Task 3: Keycloak Role Alignment
- Ensure `platform-admin`, `org-admin`, `org-member` roles exist in realm
- Auto-assign `org-member` to new users created via `mf users create`
- Auto-assign `org-admin` when user creates an org or is set as org admin
- Update `keycloak_setup.go` ConfigureRealm to create all 3 roles

### Task 4: OPA Policy Update
- Rewrite `authz.rego` with 3-tier role model
- Add resource-level rules for Settings-category resources
- Add org-scoping rules (org-admin/org-member access own org resources only)
- Test policy with different role combinations

### Task 5: Web UI Role-Based Navigation
- Add `hasRole` template function to template engine
- Update `nav.html` with conditional menu rendering based on user roles
- Pass user roles in all page data
- Add Access Denied page for unauthorized menu access attempts
- Org Admin Users & Orgs page: only show own org, hide Keycloak user management tab

### Task 6: Org-Scoped Resource Filtering
- Update app/service/secret list handlers to filter by active org
- Platform-admin sees all; org-admin/org-member see only their org resources
- Update dashboard counts to respect org scope

### Task 7: Org Admin User Permission Controls
- In Users & Orgs page, Org Admin can set per-member menu visibility
- Store member permissions in org membership (extended OrgMember struct)
- Apply member-specific permissions in nav rendering

## Files

| # | File | Action | Purpose |
|---|------|--------|---------|
| 1 | `cmd/mf/login.go` | NEW | `mf login`, `mf logout`, `mf whoami` commands |
| 2 | `cmd/mf/main.go` | MODIFY | Wire login/logout/whoami commands |
| 3 | `pkg/auth/token.go` | NEW | Token storage, refresh, and loading utilities |
| 4 | `pkg/auth/keycloak_setup.go` | MODIFY | Ensure 3 realm roles, enable direct access grants |
| 5 | `pkg/auth/policies/authz.rego` | MODIFY | Rewrite with 3-tier role model |
| 6 | `pkg/auth/middleware.go` | MODIFY | Add role extraction and org-scope context |
| 7 | `pkg/auth/opa.go` | MODIFY | Map routes to resources with role-aware input |
| 8 | `pkg/auth/org.go` | MODIFY | Add per-member permissions to OrgMember |
| 9 | `pkg/admin/server.go` | MODIFY | Pass user roles to all page templates, add hasRole func |
| 10 | `pkg/admin/static/templates/partials/nav.html` | MODIFY | Conditional menu rendering by role |
| 11 | `pkg/admin/static/templates/denied.html` | NEW | Access denied page |
| 12 | `pkg/admin/handlers.go` | MODIFY | Org-scoped resource filtering |
| 13 | `cmd/mf/users.go` | MODIFY | Attach Bearer token from `mf login` |
| 14 | `cmd/mf/orgs.go` | MODIFY | Attach Bearer token from `mf login` |

**Total: 11 files modified, 3 new files**

## Acceptance Criteria

- [ ] `mf login --username myadmin --password Admin1234` obtains and stores an access token
- [ ] `mf whoami` displays username, email, realm roles, and active org
- [ ] `mf logout` clears stored tokens
- [ ] All `mf users` and `mf orgs` CLI commands use stored Bearer token
- [ ] Platform-admin (`myadmin`) sees all menus in Web UI and all orgs
- [ ] Org-admin sees only Operations + Users & Orgs (org-scoped) in Web UI
- [ ] Org-member sees only Operations menu in Web UI
- [ ] OPA policy denies org-member/org-admin access to Settings resources
- [ ] Apps, services, secrets are filtered by active org for non-platform-admin users
- [ ] Org-admin can manage members and their permissions within their org
- [ ] `go build ./...` and `go vet ./...` pass
- [ ] Keycloak realm has `platform-admin`, `org-admin`, `org-member` roles configured

## Verification

```bash
# Build
go build ./... && go vet ./...

# CLI Login
mf login --username myadmin --password Admin1234
mf whoami
# Output: myadmin | myadmin@example.com | roles: platform-admin | org: default

# Create test users
mf users create --username orgadmin1 --email orgadmin1@test.com --password Test1234
mf users assign-role <orgadmin1-id> --role org-admin
mf users create --username dev1 --email dev1@test.com --password Test1234
mf users assign-role <dev1-id> --role org-member

# Test org-admin login
mf login --username orgadmin1 --password Test1234
mf whoami  # roles: org-admin
# Web UI: only Operations + Users & Orgs visible

# Test org-member login
mf login --username dev1 --password Test1234
mf whoami  # roles: org-member
# Web UI: only Operations visible

# Platform admin
mf login --username myadmin --password Admin1234
# Web UI: all menus visible, all orgs visible
```
