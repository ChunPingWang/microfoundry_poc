# [Epic] Enhanced Applications Page — Rich Admin Detail View

**Agent**: Analyzer | **Label**: `analyzer`

---

## Background

The current Admin Web Interface (Epic #3, PR #4) has a minimal Applications page — it shows only name, state, instance count, and routes. An administrator managing a MicroFoundry platform needs far richer visibility into deployed applications. This epic enhances the Applications menu to provide a comprehensive view across all organizations, with detailed per-application information.

## Current State Analysis

### What the App List Shows Now
| Column | Source | Detail Level |
| --- | --- | --- |
| Name | `ListApps()` → Deployment names | Minimal |
| State | RunningCount > 0 → started/stopped | Binary |
| Instances | `GetAppStatus()` → RunningCount/TotalCount | Basic |
| Routes | `ListIngresses()` → hostnames | Hostnames only |
| Actions | Scale/Delete | Basic |

### What's Missing for Admin Visibility
- **Organization/Owner**: Who deployed this app, which team owns it
- **Resource details**: Memory, CPU, disk allocation per container
- **Build/Image info**: Docker image, lifecycle type (buildpack/docker/cnb), image digest
- **Full routing**: Domain, path, protocol — not just hostname
- **Environment variables**: Configuration inspection (masked secrets)
- **Backing services**: Bound services and their status
- **Credential store**: K8s Secrets associated with the app
- **Health check config**: Probe type, endpoint, interval
- **Timestamps**: Created, last deployed, uptime
- **Container details**: Per-instance resource usage, restart count, node placement

### Data Already Available in K8s (Not Extracted)
The Kubernetes API already stores much of this data — we just don't query it:

| Data | K8s Source | Currently Extracted |
| --- | --- | --- |
| Creation time | `Deployment.Metadata.CreationTimestamp` | No |
| Memory limit | `Container.Resources.Limits.Memory` | No |
| CPU request | `Container.Resources.Requests.CPU` | No |
| Image reference | `Container.Image` | No (stored at push time but lost) |
| Image digest | `ContainerStatus.ImageID` | No |
| Env vars | `Container.Env` | No |
| Restart count | `ContainerStatus.RestartCount` | No |
| Node name | `Pod.Spec.NodeName` | No |
| Labels/Annotations | `Deployment.Metadata.Labels/Annotations` | Only basic labels |
| Ingress paths | `IngressRule.HTTP.Paths` | No (only hostname) |
| Health probe | `Container.ReadinessProbe` | No |

## Architecture Approach

### Strategy: Enrich K8s Metadata at Deploy Time, Extract at Query Time

1. **At `mf push` time**: Store rich metadata as K8s annotations on the Deployment:
   - `microfoundry.io/owner` — deploying user (from git config or flag)
   - `microfoundry.io/org` — organization (from namespace or flag)
   - `microfoundry.io/description` — app description
   - `microfoundry.io/lifecycle` — buildpack/docker/cnb
   - `microfoundry.io/buildpacks` — comma-separated buildpack names
   - `microfoundry.io/source-image` — original image ref before build
   - `microfoundry.io/services` — comma-separated bound service names

2. **At query time**: Extract data from K8s Deployment spec, Pod status, Ingress rules, and Secrets:
   - Container resources from Deployment spec
   - Runtime image from Pod's ContainerStatus
   - Full routes from Ingress rules (host + path)
   - Env vars from Container spec
   - Secrets from K8s Secrets with label `app.kubernetes.io/name={appName}`

3. **Multi-org support**: Use K8s namespaces as organizations. The admin queries across all namespaces with the `microfoundry` managed-by label.

## Task Breakdown

### Phase 1: Enhanced K8s Data Extraction
- [ ] Add `GetAppDetail()` to `pkg/k8s/app.go` — extracts full Deployment spec, annotations, container resources, env vars, probes
- [ ] Add `GetAppRoutes()` to `pkg/k8s/ingress.go` — returns full route info (host, domain, path) not just hostnames
- [ ] Add `ListAppSecrets()` to `pkg/k8s/` — finds K8s Secrets labeled for the app
- [ ] Enhance `GetAppStatus()` — add restart count, node name, image ID per instance

### Phase 2: Enhanced Data Models
- [ ] Create `AppDetail` model in `pkg/models/` — comprehensive app view struct
- [ ] Extend `InstanceStatus` with RestartCount, NodeName, ImageID
- [ ] Create `RouteDetail` model with Host, Domain, Path, Protocol
- [ ] Create `ServiceBindingInfo` model for service display
- [ ] Create `SecretInfo` model (name, type, created — no values exposed)

### Phase 3: Deploy-Time Metadata
- [ ] Enhance `DeployApp()` in `pkg/k8s/app.go` — set annotations with owner, org, lifecycle, buildpacks
- [ ] Update `cmd/mf/push.go` — pass owner info (from git config), pass services list
- [ ] Detect and store lifecycle type as annotation

### Phase 4: Enhanced Admin Handlers
- [ ] Rewrite `AppsListHandler` — show org, owner, memory, lifecycle, created time alongside existing fields
- [ ] Rewrite `AppDetailHandler` — fetch and display all enriched data
- [ ] Add `AppEnvHandler` — return environment variables (mask secrets)
- [ ] Add `AppServicesHandler` — return bound services info
- [ ] Add `AppSecretsHandler` — return secret metadata (no values)
- [ ] Update JSON API handlers to return enriched data

### Phase 5: Enhanced App List Template
- [ ] Redesign `apps.html` — add columns: Org, Owner, Memory, Lifecycle, Created
- [ ] Add filtering by org/owner/state
- [ ] Add search/filter input
- [ ] Update `app_row.html` partial with new columns

### Phase 6: Rich App Detail Template with Tabs
- [ ] Redesign `app_detail.html` with tabbed layout:
  - **Overview tab**: State, instances, routes, resources, health check, build info, timestamps
  - **Instances tab**: Enhanced instance table with restart count, node, resource usage
  - **Configuration tab**: Env vars (masked secrets), resource limits, command, labels
  - **Services tab**: Bound services list with status
  - **Routes tab**: Full route details with domain, path, protocol
  - **Logs tab**: Existing SSE log streaming (moved to tab)
- [ ] Create HTMX tab-switching partials for each tab

## Acceptance Criteria

1. App list shows: Name, Org, Owner, State, Instances, Memory, Lifecycle, Routes, Created, Actions
2. Clicking an app shows a rich detail page with tabbed sections
3. Overview tab: state, instances (running/total), all routes with domain+path, memory/CPU config, health check type, image ref, lifecycle, created/updated timestamps
4. Instances tab: index, state, uptime, restart count, node, memory/CPU usage
5. Configuration tab: env vars (SECRET values masked), resource limits, startup command, labels/annotations
6. Services tab: bound services list (or "No services bound" placeholder)
7. Routes tab: all routes with host, domain, path
8. Logs tab: live SSE streaming (existing functionality)
9. App list supports filtering by state (started/stopped/all)
10. JSON API `/api/apps/{name}` returns enriched data
11. `mf push` stores metadata annotations on K8s Deployment

## Files to Modify

| File | Change |
| --- | --- |
| `pkg/k8s/app.go` | Add `GetAppDetail()`, enhance `GetAppStatus()`, enhance `DeployApp()` |
| `pkg/k8s/ingress.go` | Add `GetAppRoutes()` returning full route details |
| `pkg/models/app.go` | Add `AppDetail`, enhance `InstanceStatus` |
| `pkg/models/route.go` | Add `RouteDetail` with full routing info |
| `cmd/mf/push.go` | Store annotations at deploy time |
| `pkg/admin/handlers.go` | Rewrite apps handlers with enriched data |
| `pkg/admin/api.go` | Update JSON API with enriched responses |
| `pkg/admin/templates.go` | Add new template helpers |

## Files to Create

| File | Description |
| --- | --- |
| `pkg/models/service.go` | ServiceBindingInfo, SecretInfo models |
| `pkg/k8s/secrets.go` | K8s Secret queries for app |
| `pkg/admin/static/templates/apps.html` | Redesigned app list |
| `pkg/admin/static/templates/app_detail.html` | Tabbed detail layout |
| `pkg/admin/static/templates/app_overview.html` | Overview tab partial |
| `pkg/admin/static/templates/app_config.html` | Configuration tab partial |
| `pkg/admin/static/templates/app_services.html` | Services tab partial |
| `pkg/admin/static/templates/app_routes.html` | Routes tab partial |
| `pkg/admin/static/templates/app_logs.html` | Logs tab partial (refactored from detail) |
