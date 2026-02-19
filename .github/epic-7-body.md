## Epic: Kubernetes Clusters Management

### Overview

Add multi-cluster Kubernetes support to MicroFoundry. Users should be able to register, manage, and deploy applications to multiple K8s clusters (Docker Desktop, EKS, GKE, AKS, native K8s) from both the CLI and the admin web interface.

### Problem

Currently, MicroFoundry is hardcoded to a single Kubernetes cluster:
- CLI commands (`push`, `apps`, `delete`, `logs`, `scale`) hardcode `docker-desktop` context
- Admin server uses a single `k8s.Client` instance
- Config supports only one `kubernetes.context` / `kubernetes.namespace`
- No way to view, add, or switch between clusters

### Requirements

1. **Admin UI: "Kubernetes Clusters" menu** — List registered clusters with provider badges, status, node count
2. **Cluster Registration** — Add clusters via admin UI (referencing kubeconfig contexts) or config file
3. **Cluster Switching** — Select active cluster from admin header dropdown; all pages operate on selected cluster
4. **Cluster Detail** — View cluster health, node pool, resource utilization, app count
5. **CLI Multi-Cluster** — `mf cluster list/add/use/remove` commands; `--cluster` flag on all commands
6. **Config Migration** — Backwards-compatible: old single-cluster config auto-migrates to new format

### Architecture

- **ClientManager** (`pkg/k8s/manager.go`) — Thread-safe client pool, lazy initialization per cluster
- **Config** — `kubernetes.clusters` map + `kubernetes.active` field in mf.yaml
- **Admin Server** — `Server.clientManager` replaces `Server.k8sClient`; all handlers use `getSelectedCluster(r)` pattern
- **Cluster Selection** — Priority: URL query param > Cookie > Config default
- **Zero new Go dependencies** — Uses existing client-go, viper, cobra

### Files to Create

| File | Description |
|------|-------------|
| `pkg/k8s/manager.go` | ClientManager with client pool |
| `pkg/admin/cluster_handlers.go` | Cluster CRUD + health check handlers |
| `pkg/admin/static/templates/clusters.html` | Cluster list page |
| `pkg/admin/static/templates/cluster_detail.html` | Cluster detail page |
| `cmd/mf/cluster.go` | `mf cluster` CLI command group |

### Files to Modify

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `Clusters` map, `Active` field, migration logic |
| `configs/mf.example.yaml` | Add `clusters:` section with examples |
| `pkg/admin/server.go` | Replace `k8sClient` with `clientManager` |
| `pkg/admin/handlers.go` | All handlers use `getSelectedCluster()` + `clientManager.GetClient()` |
| `pkg/admin/api.go` | Add `/api/clusters` endpoints |
| `pkg/admin/static/templates/partials/nav.html` | Add "Kubernetes Clusters" menu item |
| `pkg/admin/static/templates/layout.html` | Add cluster selector dropdown to header |
| `cmd/mf/push.go` | Read cluster from config (remove hardcoded context) |
| `cmd/mf/apps.go` | Read cluster from config |
| `cmd/mf/delete.go` | Read cluster from config |
| `cmd/mf/logs.go` | Read cluster from config |
| `cmd/mf/scale.go` | Read cluster from config |
| `cmd/mf/main.go` | Add `clusterCmd()` |

### Agent Workflow

- [x] Analyzer: Deep analysis (this comment)
- [ ] Data Engineer: Data model recommendations
- [ ] Product Designer: UI/UX design
- [ ] Developer: Implementation
- [ ] DevOps: Deploy and verify
- [ ] Reviewer: Code review
