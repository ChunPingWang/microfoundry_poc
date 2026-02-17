# MicroFoundry

**A micro CloudFoundry for Kubernetes** — lightweight PaaS that preserves the CloudFoundry developer experience while running on cloud-native infrastructure.

MicroFoundry replaces the heavyweight BOSH/Diego runtime with Kubernetes, managed cloud services, and modern observability. The result: `cf push`-style deployments, OSBAPI service binding, Loggregator-equivalent logging — all backed by Kubernetes, Prometheus, Loki, and Grafana Beyla.

> **Built with AI** — This project is developed through a structured Human-AI collaborative workflow using [Claude Code](https://claude.ai/claude-code). Every Epic goes through a 7-agent review process covering security, platform engineering, API design, frontend, DevOps, QA, and product management. See [How We Build](#how-we-build-human-ai-collaborative-development) for details.

---

## Highlights

- **`mf push`** — Build and deploy from source (Dockerfile or Cloud Native Buildpacks)
- **10 backing services** — MariaDB, PostgreSQL, Redis, RabbitMQ, MinIO, Kong, and more with real K8s provisioning
- **Zero-code observability** — Grafana Beyla eBPF auto-instruments all HTTP traffic for RED metrics
- **Multi-cluster** — Manage Docker Desktop, EKS, GKE, AKS clusters from a single control plane
- **Admin Dashboard** — Full-featured web UI with HTMX (no JS build step)
- **Keycloak IAM** — OIDC authentication, SCIM v2 provisioning, OPA authorization
- **MCP Server** — AI tools can deploy, scale, and manage apps via Model Context Protocol
- **Single binary** — One Go binary, no external dependencies beyond Kubernetes

---

## Admin Dashboard

The built-in admin dashboard (`mf admin`, default `:8080`) provides a complete platform management experience — application lifecycle, service catalog, multi-cluster management, observability, secrets, IAM, and platform settings in a single interface.

<p align="center">
  <img src="docs/images/dashboard-walkthrough.gif" alt="MicroFoundry Admin Dashboard Walkthrough" width="900">
</p>

<details>
<summary><strong>Screenshots</strong> (click to expand)</summary>
<br>

| Dashboard | Applications | Service Catalog |
|:---------:|:------------:|:---------------:|
| <img src="docs/images/dashboard.png" alt="Dashboard" width="280"> | <img src="docs/images/apps-list.png" alt="Applications" width="280"> | <img src="docs/images/catalog.png" alt="Catalog" width="280"> |
| Platform stats, quick links | App state, instances, routes | 10 service types, 3 plans each |

| Clusters | Monitoring & Alerts | Secrets |
|:--------:|:-------------------:|:-------:|
| <img src="docs/images/clusters.png" alt="Clusters" width="280"> | <img src="docs/images/monitoring.png" alt="Monitoring" width="280"> | <img src="docs/images/secrets.png" alt="Secrets" width="280"> |
| Multi-cluster management | Prometheus alerts, Grafana | Service & user-defined secrets |

| Users & IAM | Platform Settings | Services |
|:-----------:|:-----------------:|:--------:|
| <img src="docs/images/users-iam.png" alt="Users & IAM" width="280"> | <img src="docs/images/settings.png" alt="Settings" width="280"> | <img src="docs/images/services.png" alt="Services" width="280"> |
| Keycloak OIDC, OPA, SCIM v2 | Registry, Webhooks, SMTP | Provisioned backing services |

</details>

**Key pages:**
- **Dashboard** — Platform stats (apps, domain, namespace, K8s context) with quick links
- **Applications** — Deploy, scale, delete apps with 8-tab detail view (Overview, Instances, Config, Services, Routes, Logs, Metrics, Performance)
- **Service Catalog** — Browse 10 service types by category with plan visibility and Terraform topology editor
- **Clusters** — Register and switch between Docker Desktop, EKS, GKE, AKS clusters
- **Monitoring** — Prometheus alerts + embedded Grafana dashboards with Beyla eBPF auto-instrumentation
- **Secrets** — Service secrets (auto-created) and user-defined key-value pairs with reveal toggle
- **Users & IAM** — Keycloak user management, organizations, OPA Rego policies, audit log
- **Settings** — Container registry, webhooks, SMTP configuration stored in K8s

---

## Architecture Overview

```
                          ┌─────────────────────────────────┐
                          │         Developer / AI          │
                          └──────────┬──────────┬───────────┘
                                     │          │
                              ┌──────▽──┐  ┌────▽──────┐
                              │  CLI    │  │  MCP      │
                              │ mf push │  │  Server   │
                              │ mf logs │  │ (Claude,  │
                              │ mf bind │  │  Cursor)  │
                              └──────┬──┘  └────┬──────┘
                                     │          │
                              ┌──────▽──────────▽──────┐
                              │  MicroFoundry API +    │
                              │  Admin Dashboard (:8080)│
                              └─────────────┬──────────┘
                                            │
          ┌─────────────────────────────────┼─────────────────────────────────┐
          │                                 │                                 │
  ┌───────▽────────┐              ┌─────────▽──────────┐            ┌────────▽────────┐
  │  Build System  │              │  Kubernetes API     │            │  Observability  │
  │                │              │                     │            │                 │
  │  Dockerfile    │              │  Deployments (Apps) │            │  Prometheus     │
  │  CNB/Paketo    │              │  Services (Network) │            │  Grafana        │
  │  pack CLI      │              │  Ingress (Routes)   │            │  Loki           │
  └────────────────┘              │  Secrets (Creds)    │            │  Beyla (eBPF)   │
                                  │  ConfigMaps         │            │  AlertManager   │
                                  └─────────┬──────────┘            └─────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
            ┌───────▽───────┐       ┌───────▽───────┐      ┌───────▽───────┐
            │ API Gateway   │       │ Backing       │      │ IAM           │
            │ (Kong/Nginx/  │       │ Services      │      │ Keycloak OIDC │
            │  AWS API GW)  │       │ (DB/Cache/MQ) │      │ OPA + SCIM v2 │
            └───────────────┘       └───────────────┘      └───────────────┘
```

### Multi-Cluster Runtime

```
┌─────────────────────────────────────────────────────────────────────┐
│                     MicroFoundry Control Plane                      │
│                                                                     │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │  Config Store  │  │  Cluster     │  │  Service     │            │
│  │  (mf.yaml +   │  │  Manager     │  │  Catalog     │            │
│  │   K8s ConfigMap)│  │              │  │              │            │
│  └───────┬───────┘  └──────┬───────┘  └──────┬───────┘            │
└──────────┼─────────────────┼─────────────────┼─────────────────────┘
           │                 │                 │
    ┌──────▽──────┐   ┌──────▽──────┐   ┌──────▽──────┐
    │ Docker      │   │ AWS EKS     │   │ GCP GKE     │
    │ Desktop     │   │             │   │             │
    │ cf-local.dev│   │ *.app.com   │   │ *.app.io    │
    └─────────────┘   └─────────────┘   └─────────────┘
```

---

## CloudFoundry Mapping

| CF Component | MicroFoundry Equivalent | Implementation |
|---|---|---|
| **Diego Cell** | Kubernetes Pod | K8s Deployment + Service + Ingress |
| **Gorouter** | API Gateway (Kong / Nginx / AWS API GW) | Pluggable ingress controller |
| **Cloud Controller** | MicroFoundry API Server (Go) | Lightweight API → K8s API directly |
| **Buildpacks** | Cloud Native Buildpacks (CNB/Paketo) | Source-to-container without Dockerfile |
| **Service Broker** | Built-in K8s-native Broker | 10 service types with real K8s provisioning |
| **Service Catalog** | Built-in Catalog (10 services) | MariaDB, PostgreSQL, Redis, RabbitMQ, MinIO, Kong, etc. |
| **Loggregator** | Promtail + Loki | Log collection per pod → Loki aggregation |
| **Doppler/Metrics** | Prometheus + Grafana + Beyla eBPF | Auto-instrumented RED metrics |
| **NATS (Alerts)** | AlertManager | Prometheus alerting rules + AlertManager |
| **Config Server** | K8s Secrets + ConfigMaps | VCAP_SERVICES injection + platform settings |
| **UAA** | Keycloak OIDC + OPA + SCIM v2 | OIDC auth, Rego policies, standard provisioning |
| **CF CLI** | `mf` CLI (Cobra) | 20+ commands mirroring CF CLI |
| **— (new)** | MCP Server | AI tool integration via Model Context Protocol |

---

## Developer Experience

### CLI Commands

| Command | CF Equivalent | Description |
|---|---|---|
| `mf push [app]` | `cf push` | Build + deploy from source (Dockerfile or CNB) |
| `mf apps` | `cf apps` | List deployed applications with status |
| `mf app [name]` | `cf app` | Show app details and instance list |
| `mf logs [app]` | `cf logs` | Stream or fetch application logs |
| `mf scale [app] -i N` | `cf scale` | Scale application instances |
| `mf delete [app]` | `cf delete` | Delete app and clean up routes |
| `mf catalog` | `cf marketplace` | List available services by category |
| `mf create-service <type> <plan> <name>` | `cf create-service` | Provision a backing service |
| `mf services` | `cf services` | List provisioned service instances |
| `mf bind-service <app> <svc>` | `cf bind-service` | Bind service → inject VCAP_SERVICES |
| `mf unbind-service <app> <svc>` | `cf unbind-service` | Unbind service from app |
| `mf delete-service <name>` | `cf delete-service` | Delete service instance |
| `mf secrets` | — | List managed secrets |
| `mf create-secret <name> k=v...` | — | Create user-defined secret |
| `mf delete-secret <name>` | — | Delete a secret |
| `mf admin` | — | Start web dashboard (:8080) |
| `mf setup keycloak` | — | Deploy Keycloak for authentication |
| `mf setup keycloak-realm` | — | Configure Keycloak realm and client |
| `mf setup keycloak-idp` | — | Add identity provider (Google, GitHub, Amazon) |
| `mf version` | — | Print version |

### MCP Server Tools

| MCP Tool | Description |
|---|---|
| `mf_push` | Deploy an application from source |
| `mf_logs` | Stream or fetch application logs |
| `mf_bind_service` | Bind a backing service to an app |
| `mf_create_service` | Provision a backing service instance |
| `mf_routes` | List or manage application routes |
| `mf_scale` | Scale app instances or resources |
| `mf_env` | View or set environment variables |
| `mf_apps` | List deployed applications |
| `mf_delete` | Remove a deployed application |

---

## Platform Capabilities

### Application Lifecycle

```bash
$ mf push hello-world
Building image...         [Dockerfile detected]
Deploying to K8s...       [deployment/hello-world created]
Creating ingress route... [hello-world.cf-local.dev → :8080]
Updating hosts file...    [127.0.0.1 hello-world.cf-local.dev]
Waiting for rollout...    [3/3 instances running]

   app:        hello-world
   url:        http://hello-world.cf-local.dev
   instances:  3/3 running
   memory:     256M
   metrics:    auto-instrumented (Beyla eBPF)
   dashboard:  http://localhost:8080/apps/hello-world?tab=performance
```

### Service Broker & Catalog

```
┌─────────────────────────────────────────────────────────────┐
│                    Service Catalog                           │
├──────────────┬──────────────┬───────────────┬───────────────┤
│  Databases   │  Caches      │  Messaging    │  Storage/GW   │
├──────────────┼──────────────┼───────────────┼───────────────┤
│  MariaDB     │  Redis       │  RabbitMQ     │  MinIO (S3)   │
│  PostgreSQL  │  Memcached   │  ActiveMQ     │  Kong          │
│  ClickHouse  │              │               │  Nginx         │
└──────────────┴──────────────┴───────────────┴───────────────┘
  Each with 3 plans: small (dev) / medium (staging) / large (prod)
```

```bash
$ mf catalog
$ mf create-service postgresql small my-db
$ mf bind-service hello-world my-db    # Injects VCAP_SERVICES
```

### Observability Stack

```
┌──────────────────────────────────────────────────────────────────┐
│  Observability Stack                                              │
│                                                                   │
│  ┌──────────┐    eBPF     ┌────────────┐   scrape  ┌──────────┐ │
│  │  App Pod  │◀───────────│  Beyla      │──────────▶│Prometheus│ │
│  │ (any lang)│  zero-code │  DaemonSet  │           │          │ │
│  └──────────┘  intercept  └────────────┘           └────┬─────┘ │
│       │                                                  │       │
│       │ stdout/stderr      ┌────────────┐         ┌─────▽─────┐ │
│       └───────────────────▶│  Promtail  │────────▶│  Grafana  │ │
│                            └────────────┘         │  + Loki   │ │
│                                                   └─────┬─────┘ │
│  Pre-computed RED Metrics:                              │       │
│    microfoundry:http_request_rate:5m               ┌────▽──────┐│
│    microfoundry:http_error_rate:5m                 │AlertManager││
│    microfoundry:http_latency_p50/p95/p99:5m        └───────────┘│
└──────────────────────────────────────────────────────────────────┘
```

### Authentication & Authorization

```bash
mf setup keycloak                # Deploy Keycloak to cluster
mf setup keycloak-realm          # Configure realm, client, roles
mf setup keycloak-idp \          # Add social login
  --provider google \
  --client-id <ID> --client-secret <SECRET>
```

- **OIDC Authorization Code Flow** with PKCE
- **Social login**: Google, GitHub, Amazon identity providers
- **Roles**: platform-admin, org-admin, org-member, viewer
- **Organizations**: Multi-tenant org management with member invitations
- **OPA Authorization**: Embedded Open Policy Agent with Rego policies
- **SCIM v2**: Standard identity provisioning endpoints (RFC 7643/7644)
- **Audit Log**: In-memory ring buffer with resource/action/decision tracking

---

## Getting Started

### Prerequisites

- Go 1.25+
- Docker Desktop with Kubernetes enabled
- kubectl
- Helm 3

### Build & Install

```bash
make build              # Build to bin/mf
make install            # Install to GOPATH/bin
```

### Deploy Monitoring Stack

```bash
make monitoring-install  # Prometheus + Grafana + Loki + Beyla
```

### Deploy Your First App

```bash
mf push hello-world                          # Deploy from source
mf create-service postgresql small my-db     # Provision a database
mf bind-service hello-world my-db            # Bind DB → app
mf logs hello-world                          # Stream logs
mf admin                                     # Open web dashboard
```

### Set Up Authentication (Optional)

```bash
mf setup keycloak                            # Deploy Keycloak
kubectl port-forward -n microfoundry svc/keycloak 8180:8180
mf setup keycloak-realm --url http://localhost:8180
# Add auth section to configs/mf.yaml, then restart mf admin
```

### Configuration

```bash
cp configs/mf.example.yaml configs/mf.yaml
```

```yaml
kubernetes:
  active: "docker-desktop"
  clusters:
    docker-desktop:
      context: "docker-desktop"
      namespace: "microfoundry"
      domain: "cf-local.dev"
      provider: "docker-desktop"
    eks-prod:
      context: "arn:aws:eks:us-west-2:..."
      namespace: "microfoundry"
      domain: "apps.example.com"
      provider: "eks"
```

---

## Tech Stack

| Layer | Technology | Purpose |
|---|---|---|
| **Language** | Go 1.25 | API server, CLI, MCP server, all controllers |
| **CLI** | Cobra + Viper | Command parsing + configuration |
| **Runtime** | Kubernetes | Application scheduling and orchestration |
| **Build** | Cloud Native Buildpacks (Paketo) | Source-to-container builds |
| **Ingress** | Kong / Nginx | API gateway and routing |
| **Metrics** | Prometheus + Grafana | Collection and visualization |
| **Logs** | Promtail + Loki | Aggregation and querying |
| **Auto-Instrumentation** | Grafana Beyla (eBPF) | Zero-code HTTP metrics |
| **Alerting** | AlertManager | Alert routing and notification |
| **Authentication** | Keycloak + go-oidc | OIDC authorization code flow |
| **Authorization** | Open Policy Agent (OPA) | Rego-based policy evaluation |
| **Provisioning** | SCIM v2 | Standard identity provisioning |
| **Sessions** | gorilla/sessions | Cookie-based session management |
| **UI** | Go templates + HTMX + Tailwind CSS | Server-side rendering, no JS build step |
| **IaC** | Terraform | Cloud resource topology management |
| **AI** | Model Context Protocol (MCP) | AI tool platform access |
| **K8s Client** | client-go | Kubernetes API interactions |

---

## Project Structure

```
microfoundry/
├── cmd/mf/                    # CLI entry points (20+ commands)
│   ├── main.go                #   root command + version
│   ├── push.go                #   mf push (build → registry → deploy → ingress → hosts)
│   ├── apps.go                #   mf apps / mf app
│   ├── logs.go                #   mf logs (stream + history)
│   ├── scale.go               #   mf scale
│   ├── delete.go              #   mf delete
│   ├── catalog.go             #   mf catalog (marketplace)
│   ├── create_service.go      #   mf create-service
│   ├── services.go            #   mf services / mf service
│   ├── bind_service.go        #   mf bind-service
│   ├── unbind_service.go      #   mf unbind-service
│   ├── delete_service.go      #   mf delete-service
│   ├── secrets.go             #   mf secrets / mf create-secret / mf delete-secret
│   ├── admin.go               #   mf admin (web dashboard + auth + OPA init)
│   └── setup.go               #   mf setup keycloak / keycloak-realm / keycloak-idp
│
├── pkg/                       # Go packages
│   ├── admin/                 #   Web dashboard + API handlers (100+ routes)
│   │   ├── server.go          #     HTTP server, route registration, auth/OPA middleware
│   │   ├── handlers.go        #     App detail, dashboard, tab routing
│   │   ├── api.go             #     JSON API endpoints
│   │   ├── scim_handlers.go   #     SCIM v2 endpoints (RFC 7643/7644)
│   │   ├── iam_handlers.go    #     IAM tab + Keycloak user management
│   │   ├── org_handlers.go    #     Organization + member management
│   │   ├── performance_handlers.go  # RED metrics + observability
│   │   ├── service_handlers.go      # Service management UI
│   │   ├── cluster_handlers.go      # Multi-cluster management
│   │   ├── monitoring_handlers.go   # Alert & monitoring UI
│   │   ├── secret_handlers.go       # Secret management UI
│   │   ├── settings_handlers.go     # Registry, webhooks, SMTP config
│   │   ├── topology_handlers.go     # Terraform topology editor
│   │   ├── logs.go            #     SSE log streaming
│   │   ├── templates.go       #     Template renderer (clone pattern)
│   │   └── static/            #     Embedded HTML/CSS/JS templates
│   │       ├── templates/     #       20+ page templates
│   │       │   ├── partials/  #       Shared partials (nav, header)
│   │       │   └── tabs/      #       HTMX tab partials (12 tabs)
│   │       └── css/           #       Tailwind CSS
│   ├── auth/                  #   Authentication & Authorization
│   │   ├── oidc.go            #     OIDC authorization code flow with PKCE
│   │   ├── session.go         #     Cookie-based session management
│   │   ├── keycloak.go        #     Realm/client/IdP setup
│   │   ├── keycloak_admin.go  #     Keycloak Admin REST API client (user CRUD)
│   │   ├── scim.go            #     SCIM v2 types + Keycloak conversion
│   │   ├── opa.go             #     Embedded OPA engine (Rego evaluation)
│   │   ├── opa_middleware.go  #     OPA HTTP middleware + route classification
│   │   ├── audit.go           #     Authorization audit log (ring buffer)
│   │   ├── org.go             #     Organization & member management
│   │   ├── middleware.go      #     InjectUser middleware
│   │   ├── config.go          #     Auth configuration types
│   │   └── policies/          #     Embedded Rego policies
│   │       └── authz.rego     #       Default authorization policy
│   ├── build/                 #   Source-to-image (Dockerfile + CNB + registry push)
│   ├── config/                #   Multi-cluster configuration (Viper + YAML)
│   ├── hosts/                 #   /etc/hosts management
│   ├── k8s/                   #   Kubernetes client + operations
│   │   ├── client.go          #     K8s API client wrapper
│   │   ├── app.go             #     Deployment/Service/Pod management
│   │   ├── ingress.go         #     Ingress route management
│   │   ├── manager.go         #     Multi-cluster ClientManager
│   │   ├── registry.go        #     imagePullSecret management
│   │   └── keycloak.go        #     Keycloak K8s deployment
│   ├── manifest/              #   CF manifest.yml parser
│   ├── models/                #   Core data models
│   ├── monitoring/            #   Observability stack integration
│   │   ├── prometheus.go      #     Prometheus query client (RED metrics)
│   │   ├── grafana.go         #     Grafana dashboard URL builder
│   │   ├── loki.go            #     Log aggregation client
│   │   ├── alertmanager.go    #     Alert management client
│   │   ├── metrics.go         #     Custom Prometheus metrics
│   │   ├── collector.go       #     Background metrics collection (30s)
│   │   └── middleware.go      #     HTTP metrics middleware
│   ├── secrets/               #   K8s Secret management
│   ├── service/               #   Service broker + catalog + provisioning
│   │   ├── catalog.go         #     10 service types, 3 plans each
│   │   ├── provisioner.go     #     K8s-native provisioning (StatefulSet, PVC)
│   │   ├── binder.go          #     VCAP_SERVICES injection
│   │   └── visibility.go      #     Plan visibility toggle
│   ├── settings/              #   Platform settings (ConfigMap/Secret store)
│   └── terraform/             #   Terraform topology management
│
├── deploy/
│   ├── k8s/                   # Kubernetes manifests
│   │   ├── base/              #   Base manifests (namespace)
│   │   └── overlays/          #   Kustomize overlays (local, EKS, GKE, AKS)
│   └── monitoring/            # Observability stack
│       ├── install.sh         #   One-command monitoring setup
│       ├── beyla-config.yaml  #   Beyla eBPF DaemonSet
│       ├── prometheus-recording-rules.yaml
│       ├── dashboards/        #   Grafana dashboards
│       └── alerts/            #   Prometheus alerting rules
│
├── test/                      # E2E tests (Playwright)
│   ├── playwright.config.ts   #   Test configuration
│   ├── e2e/                   #   82 test cases across 8 suites
│   └── helpers/               #   Test utilities
│
├── configs/mf.example.yaml   # Example configuration
├── docs/                      # Documentation
│   ├── user-manual.md         #   Complete user guide
│   ├── architecture.md        #   Technical architecture
│   ├── admin-guide.md         #   Admin dashboard guide
│   ├── cloudfoundry-architecture.md  # CF reference
│   └── observability-capacity.md     # Monitoring docs
├── Makefile                   # Build targets
├── Dockerfile                 # Container build
├── CLAUDE.md                  # Agent workflow rules
└── LICENSE
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [User Manual](docs/user-manual.md) | Complete guide to deploying and managing applications |
| [Architecture](docs/architecture.md) | Technical architecture and component design |
| [Admin Guide](docs/admin-guide.md) | Admin dashboard pages, API reference (100+ endpoints) |
| [CF Architecture](docs/cloudfoundry-architecture.md) | CloudFoundry reference architecture |
| [Observability](docs/observability-capacity.md) | Monitoring stack documentation |

---

## How We Build: Human-AI Collaborative Development

MicroFoundry is developed through a structured **Human-AI pair programming workflow** using [Claude Code](https://claude.ai/claude-code) (Anthropic's AI coding agent). This isn't casual AI assistance — it's a formalized development process where AI participates in every phase of the software development lifecycle.

### The Workflow

```
┌────────────────────────────────────────────────────────────────────┐
│                    Epic Development Lifecycle                       │
│                                                                    │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    │
│  │ Analyzer  │───▶│  Issue   │───▶│  Agent   │───▶│   Plan   │    │
│  │  Check   │    │ Creation │    │Discussion│    │  Review  │    │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘    │
│       │                                                │          │
│       │  ┌──────────────────────────────────────────────┘          │
│       │  │                                                        │
│  ┌────▽──▽───┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │   Code    │───▶│    PR    │───▶│  Agent   │───▶│  Merge   │  │
│  │  Implement│    │ Creation │    │  Review  │    │  to rc   │  │
│  └───────────┘    └──────────┘    └──────────┘    └──────────┘  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

**Each Epic follows this process:**

1. **Analyzer Check** — Verify no open PR dependencies; ensure `rc` branch is clean
2. **Issue Creation** — Create a GitHub issue with full Epic scope, architecture decisions, and file plan
3. **Agent Discussion** — 7 specialized agents comment on the issue with their domain expertise
4. **Plan Review** — Human reviews agent feedback and approves the implementation plan
5. **Implementation** — Claude Code implements all code changes (typically 10-25 files per Epic)
6. **PR Creation** — Create a pull request targeting the `rc` branch
7. **Agent Review** — All 7 agents post review comments on the PR with findings and recommendations
8. **Merge** — Human reviews and merges to `rc`

### The 7+1 Agent Personas

Every issue and PR receives comments from **7 specialized review agents**. Additionally, a **Document Expert** agent runs periodically to keep docs in sync.

#### Per-Epic Review Agents (every Epic)

| Agent | Role | Focus Area |
|-------|------|------------|
| **Security Architect** | Identifies vulnerabilities, auth gaps, injection risks | OWASP, authentication flows, secrets management, policy bypass |
| **Platform Engineer** | Reviews reliability, performance, crash safety | Nil guards, error handling, middleware ordering, resource leaks |
| **API Designer** | Ensures API consistency, spec compliance, contracts | REST conventions, SCIM RFC compliance, pagination, error schemas |
| **Frontend Engineer** | Reviews UI/UX, template patterns, accessibility | HTMX interactions, Tailwind consistency, error states, responsive design |
| **DevOps Engineer** | Evaluates deployment safety, observability, CI/CD | Rolling updates, log formats, monitoring integration, build pipeline |
| **QA Engineer** | Designs test plans, verification matrices, regression checks | Unit/integration/manual test cases, edge cases, acceptance criteria |
| **Product Manager** | Assesses scope, user impact, prioritization | User stories, release blocking decisions, success metrics, communication |

#### Periodic Batch Agent (every 5 Epics)

| Agent               | Role                                         | Focus Area                                                             |
|---------------------|----------------------------------------------|------------------------------------------------------------------------|
| **Document Expert** | Syncs README and docs/ with codebase reality | CLI commands, API endpoints, config changes, admin pages, architecture |

The Document Expert activates after every 5th merged Epic. It audits all changes since the last sync, classifies their documentation impact (Critical → None), updates README.md and all 5 docs files, then creates a docs-only PR. This ensures documentation never drifts more than 5 Epics behind the actual codebase. Full workflow spec: [`.github/agents/doc-expert.md`](.github/agents/doc-expert.md).

### Branch Strategy: Release-Candidate Flow

We use a **three-tier branching model** designed for structured integration and safe promotion:

```
main (stable release — production-ready)
  └── rc (release-candidate — integration & validation)
        ├── epic/feature-a  →  PR targets rc
        ├── epic/feature-b  →  PR targets rc
        └── epic/feature-c  →  PR targets rc
```

| Branch | Purpose | Merges From | Merges To |
|--------|---------|-------------|-----------|
| **`main`** | Stable release. Always deployable. Tagged for releases. | `rc` only | — |
| **`rc`** | Integration branch. All Epics land here first. Validated before promoting to `main`. | `epic/*` branches | `main` |
| **`epic/*`** | Feature branches. One per Epic. Created from `rc`, never from `main` or another epic. | — | `rc` |

**Key rules:**

1. **All PRs target `rc`**, never `main` directly
2. **Epic branches are independent** — never stack PRs by merging one epic into another
3. **Analyzer checks dependencies** — before starting a new Epic, verify no open PRs conflict
4. **`rc` → `main` promotion** happens when a set of features is validated (build passes, agents reviewed, human approved)
5. **Fast-forward merges preferred** for `rc` → `main` to keep clean history

**Typical flow:**

```
git checkout rc && git pull origin rc
git checkout -b epic/new-feature        # Branch from rc
# ... implement ...
gh pr create --base rc                  # PR targets rc
# ... agent review + human merge ...
# When ready to release:
git checkout main && git merge rc       # Promote to main
```

### Why This Workflow?

This structured process serves several purposes:

- **Quality through diverse perspectives** — Each agent catches issues specific to their domain. Security finds auth bypasses, QA designs test matrices, DevOps flags deployment risks — simultaneously.
- **Documented decision trail** — Every architectural decision, trade-off, and review finding is captured in GitHub issues and PR comments, creating a permanent knowledge base.
- **Consistent velocity** — Epics averaging 15-25 files are implemented, reviewed, and merged in single sessions with comprehensive coverage.
- **Human oversight** — The human developer ([@younjinjeong](https://github.com/younjinjeong)) reviews all agent feedback, approves plans, and makes final merge decisions. AI proposes; human disposes.

---

## Development History

MicroFoundry has been built incrementally through a series of Epics, each adding a major capability:

| PR | Epic | Description | Files |
|----|------|-------------|-------|
| [#2](https://github.com/younjinjeong/microfoundry/pull/2) | Local K8s Runtime | `mf push`, `mf apps`, `mf logs`, `mf scale`, `mf delete` — core CLI | — |
| [#4](https://github.com/younjinjeong/microfoundry/pull/4) | Admin Dashboard | Web admin interface with HTMX, dashboard, app management | — |
| [#6](https://github.com/younjinjeong/microfoundry/pull/6) | App Detail View | 8-tab application detail view (Overview → Performance) | — |
| [#8](https://github.com/younjinjeong/microfoundry/pull/8) | Multi-Cluster | Multi-cluster Kubernetes management with ClientManager | — |
| [#10](https://github.com/younjinjeong/microfoundry/pull/10) | Backing Services | Service catalog, provisioning, binding, VCAP_SERVICES | — |
| [#11](https://github.com/younjinjeong/microfoundry/pull/11) | Real K8s Provisioning | StatefulSet + PVC provisioning for all 10 service types | — |
| [#13](https://github.com/younjinjeong/microfoundry/pull/13) | Terraform Topologies | Terraform-based service topology management | — |
| [#15](https://github.com/younjinjeong/microfoundry/pull/15) | Catalog & Visibility | Service catalog browser with plan visibility controls | — |
| [#17](https://github.com/younjinjeong/microfoundry/pull/17) | Monitoring & Logging | Prometheus + Loki + Grafana + AlertManager integration | — |
| [#20](https://github.com/younjinjeong/microfoundry/pull/20) | Beyla eBPF | Netflix Atlas-inspired auto-instrumentation with Beyla | — |
| [#22](https://github.com/younjinjeong/microfoundry/pull/22) | Observability Hardening | Security, resilience & capacity for monitoring stack | — |
| [#24](https://github.com/younjinjeong/microfoundry/pull/24) | Platform Config | Registry, webhooks, SMTP settings via admin UI | — |
| [#26](https://github.com/younjinjeong/microfoundry/pull/26) | Keycloak UAA | OIDC authentication with Keycloak, sessions, org management | — |
| [#31](https://github.com/younjinjeong/microfoundry/pull/31) | E2E Testing | Playwright E2E test suite (82 test cases, 8 suites) | — |
| [#34](https://github.com/younjinjeong/microfoundry/pull/34) | IAM & SCIM | Keycloak user CRUD, SCIM v2, OPA authorization, audit log | 22 |
| [#37](https://github.com/younjinjeong/microfoundry/pull/37) | IAM Hardening | Authz bypass fix, error handling, SCIM compliance, OPA atomicity | 7 |

### External Contributions

| PR | Author | Description |
|----|--------|-------------|
| [#29](https://github.com/younjinjeong/microfoundry/pull/29) | [@byunjuneseok](https://github.com/byunjuneseok) | Fix: add EnsureNamespace to create-service command |

---

## Contributing

Contributions are welcome! Here's how the project operates:

### For Human Contributors

1. Fork the repository
2. Create a feature branch from `rc` (not `main`)
3. Make your changes
4. Submit a PR targeting `rc`
5. The maintainer will review with agent assistance

### For AI-Assisted Development

This project uses Claude Code with a defined workflow in [CLAUDE.md](CLAUDE.md). If you're experimenting with AI-assisted development:

1. The agent workflow rules in `CLAUDE.md` define branch strategy and conventions
2. Each Epic follows the Analyzer → Issue → Discussion → Plan → Implement → PR → Review cycle
3. 7 agent personas provide structured feedback on issues and PRs
4. All changes must pass `go build ./...` and `go vet ./...`

### Build & Verify

```bash
go build ./...          # Must pass
go vet ./...            # Must pass
go test ./...           # Run tests
make e2e                # Run Playwright E2E tests
```

---

## License

See [LICENSE](LICENSE).
