# [Epic] MicroFoundry Web Admin Interface

**Agent**: Analyzer | **Label**: `analyzer`

---

## Background

MicroFoundry currently operates as a CLI-only platform (`mf push`, `mf apps`, `mf scale`, etc.). While the CLI is effective for developers deploying applications, platform administrators need a visual interface to monitor and manage the entire MicroFoundry environment. This epic adds a **Web Admin Interface** that provides centralized visibility into platform configuration, deployed applications, backing services, secrets, and user/org management.

## Objective

Build a server-side rendered web dashboard embedded in the `mf` binary that allows administrators to:

1. View all platform configuration (domain, namespace, K8s context, GitHub settings)
2. List and manage deployed applications (status, scale, delete, view logs)
3. View backing services configuration (placeholder for future)
4. Manage secrets storage (placeholder for future)
5. Administer organizations and users (placeholder for future)

## Architecture Analysis

### Existing Infrastructure to Reuse

| Package | Methods Available | Admin Use |
| --- | --- | --- |
| `pkg/k8s/app.go` | `ListApps()`, `GetAppStatus()`, `ScaleApp()`, `DeleteApp()`, `GetAppLogs()` | Apps page, API endpoints, log streaming |
| `pkg/k8s/ingress.go` | `ListIngresses()`, `CreateIngress()`, `DeleteIngress()` | Route display on apps page |
| `pkg/k8s/client.go` | `NewClient()`, `EnsureNamespace()` | Admin server K8s connection |
| `pkg/config/config.go` | `Load()`, `Config`, `KubernetesConfig`, `GitHubConfig` | Configuration page |
| `pkg/models/` | `App`, `AppStatus`, `InstanceStatus`, `Route`, `HealthCheck` | JSON-tagged structs for API serialization |

### Technology Decision

| Choice | Technology | Rationale |
| --- | --- | --- |
| HTTP Server | Go `net/http` (stdlib) | No new dependencies, Go 1.22+ pattern routing |
| Templating | Go `html/template` + `embed` | SSR, single binary, zero build step |
| Interactivity | HTMX 2.x | Partial page updates without SPA framework |
| Styling | Tailwind CSS (CDN) | Zero build step, utility-first CSS |
| Log Streaming | Server-Sent Events (SSE) | Maps directly to `GetAppLogs()` io.ReadCloser |
| Entry Point | `mf admin` CLI command | Embedded in existing binary, same K8s client |

### Why Not a SPA (React/Vue)?

- Adds Node.js build dependency to a Go project
- Separate build pipeline, separate Docker stage
- Admin dashboard doesn't need complex client-side state
- Go templates + HTMX achieves the same UX with zero JS build chain

## Task Breakdown

### Phase 1: Server Skeleton
- [ ] Create `cmd/mf/admin.go` — Cobra command with `--port` and `--host` flags
- [ ] Create `pkg/admin/server.go` — HTTP server, route registration, `ListenAndServe()`
- [ ] Create `pkg/admin/templates.go` — `embed.FS` directives, template functions, renderer
- [ ] Create base `layout.html` template with Tailwind CDN + HTMX
- [ ] Create `dashboard.html` with summary cards
- [ ] Register `adminCmd()` in `cmd/mf/main.go`

### Phase 2: Applications Pages
- [ ] Create `handlers.go` with `DashboardHandler`, `AppsListHandler`, `AppDetailHandler`
- [ ] Create `apps.html` — table with state badges, routes, action buttons
- [ ] Create `app_detail.html` — info, instances table, log panel
- [ ] Create sidebar navigation partial (`nav.html`)
- [ ] Create page header partial (`header.html`)

### Phase 3: HTMX Interactivity
- [ ] Add `ScaleAppHandler` — form POST, returns updated row partial
- [ ] Add `DeleteAppHandler` — DELETE with confirm, removes row
- [ ] Create `app_row.html` — inline scale form + delete button
- [ ] Create `app_instances.html` — instance status with 5s polling
- [ ] Add auto-refresh on app list (10s interval)

### Phase 4: Log Streaming
- [ ] Create `pkg/admin/logs.go` — SSE handler bridging `GetAppLogs()` to event-stream
- [ ] Add log panel to `app_detail.html` with HTMX SSE extension
- [ ] Start/stop toggle for log streaming

### Phase 5: Config, Placeholders, JSON API
- [ ] Create `config.html` — K8s context, namespace, domain, GitHub settings
- [ ] Create `services.html` — "Backing Services — Coming Soon"
- [ ] Create `secrets.html` — "Secrets Management — Coming Soon"
- [ ] Create `users.html` — "Users & Organizations — Coming Soon"
- [ ] Create `pkg/admin/api.go` — JSON endpoints: `/api/apps`, `/api/apps/{name}`, `/api/config`
- [ ] Update `configs/mf.example.yaml` with admin section

## Acceptance Criteria

1. `go build ./cmd/mf` succeeds with zero new Go dependencies
2. `mf admin` starts server at `http://localhost:8080`
3. Dashboard shows real-time app count, domain, namespace, K8s context
4. `/apps` lists deployed applications with live status from K8s
5. Scale/delete work via HTMX without full page reload
6. `/apps/{name}` shows per-instance status + live log streaming via SSE
7. `/config` displays all platform configuration
8. `/api/apps` returns JSON array of app status objects
9. Placeholder pages render for services, secrets, users
10. All HTML templates embedded in binary via Go `embed`

## File Structure

```
pkg/admin/
├── server.go                           # HTTP server + route registration
├── handlers.go                         # SSR page handlers
├── api.go                              # JSON API handlers
├── logs.go                             # SSE log streaming
├── templates.go                        # embed.FS + template renderer
└── static/
    ├── css/custom.css
    └── templates/
        ├── layout.html                 # Base layout (Tailwind + HTMX)
        ├── dashboard.html              # Summary cards
        ├── apps.html                   # App list table
        ├── app_detail.html             # App detail + logs
        ├── app_row.html                # HTMX partial: table row
        ├── app_instances.html          # HTMX partial: instance table
        ├── scale_form.html             # HTMX partial: scale form
        ├── config.html                 # Configuration page
        ├── services.html               # Placeholder
        ├── secrets.html                # Placeholder
        ├── users.html                  # Placeholder
        └── partials/
            ├── nav.html                # Sidebar navigation
            └── header.html             # Page header
```

## API Endpoints

| Method | Path | Type | Description |
| --- | --- | --- | --- |
| GET | `/` | HTML | Dashboard |
| GET | `/apps` | HTML | App list (auto-refresh 10s) |
| GET | `/apps/{name}` | HTML | App detail |
| GET | `/apps/{name}/instances` | HTML | HTMX partial (poll 5s) |
| GET | `/apps/{name}/logs/stream` | SSE | Live log stream |
| POST | `/apps/{name}/scale` | HTML | Scale action (HTMX) |
| DELETE | `/apps/{name}` | HTML | Delete action (HTMX) |
| GET | `/config` | HTML | Configuration |
| GET | `/services` | HTML | Placeholder |
| GET | `/secrets` | HTML | Placeholder |
| GET | `/users` | HTML | Placeholder |
| GET | `/api/apps` | JSON | List apps |
| GET | `/api/apps/{name}` | JSON | App detail |
| GET | `/api/config` | JSON | Platform config |
| POST | `/api/apps/{name}/scale` | JSON | Scale app |
| DELETE | `/api/apps/{name}` | JSON | Delete app |
