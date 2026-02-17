# MicroFoundry

A micro CloudFoundry for Kubernetes — lightweight implementation of CF core concepts with cloud-native infrastructure.

MicroFoundry preserves the CloudFoundry developer experience (`cf push`, service binding, route management, `cf logs`) while replacing the heavyweight BOSH/Diego infrastructure with Kubernetes, managed cloud services, and modern observability. The goal is **simplicity**: instead of operating dozens of CF components, MicroFoundry delegates to Kubernetes primitives and CSP-native services.

---

## Design Objectives

1. **Lightweight CloudFoundry Core** — Implement the essential CloudFoundry concepts (Buildpack, Service Broker, Service Catalog, Loggregator, Metrics & Alerts, CLI) without the operational complexity of full CF deployment.

2. **Full CF Developer Experience** — Buildpack-based builds, OSBAPI-compatible Service Broker, Service Catalog with plans, Loggregator-style log aggregation, Prometheus/Grafana metrics and alerting, and `cf push`-style CLI.

3. **Multi-Cloud Kubernetes Runtime** — Use Kubernetes from every major CSP (EKS, GKE, AKS) and on-premise (Docker Desktop, bare-metal) as the universal runtime. No VM-based deployment.

4. **Network-Aware Service Integration** — Handle network allocation between Kubernetes clusters and backing services (databases, caches, message queues) with proper service discovery and credential injection.

5. **Pluggable API Gateway Routing** — Place API Gateways (AWS API Gateway, Nginx, Kong, etc.) in the network routing path for endpoint access, rate limiting, TLS termination, and authentication.

6. **AI-Native Platform Access** — Expose the full platform as an MCP (Model Context Protocol) server and support AI Agent workloads, enabling AI tools to deploy, manage, and monitor applications directly.

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
            │ API Gateway   │       │ Backing       │      │ Secret        │
            │ (Kong/Nginx/  │       │ Services      │      │ Store         │
            │  AWS API GW)  │       │ (DB/Cache/MQ) │      │ (K8s Secrets) │
            └───────────────┘       └───────────────┘      └───────────────┘
```

### Multi-Cluster Kubernetes Runtime

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

## CloudFoundry → MicroFoundry Mapping

| CF Component | MicroFoundry Equivalent | Implementation |
|---|---|---|
| **Diego Cell** | Kubernetes Pod | K8s Deployment + Service + Ingress |
| **Gorouter** | API Gateway (Kong / Nginx / AWS API GW) | Pluggable ingress controller in routing path |
| **Cloud Controller** | MicroFoundry API Server (Go) | Lightweight API → K8s API directly |
| **Buildpacks** | Cloud Native Buildpacks (CNB/Paketo) | Source-to-container without Dockerfile |
| **Service Broker** | Built-in K8s-native Broker | 10 service types with real K8s provisioning |
| **Service Catalog** | Built-in Catalog (10 services) | MariaDB, PostgreSQL, Redis, RabbitMQ, MinIO, Kong, etc. |
| **Loggregator** | Promtail + Loki | Log collection per pod → Loki aggregation |
| **Doppler/Metrics** | Prometheus + Grafana + Beyla eBPF | Auto-instrumented RED metrics (Netflix Atlas-inspired) |
| **NATS (Alerts)** | AlertManager | Prometheus alerting rules + AlertManager |
| **Config Server** | K8s Secrets + ConfigMaps | VCAP_SERVICES injection + platform settings |
| **UAA** | Keycloak OIDC + gorilla/sessions | Authorization code flow with PKCE, social login |
| **Blobstore** | S3 / GCS / MinIO | Managed object storage for artifacts |
| **CF CLI** | `mf` CLI (Cobra) | 20+ commands mirroring CF CLI |
| **— (new)** | MCP Server | AI tool integration via Model Context Protocol |

See [docs/cloudfoundry-architecture.md](docs/cloudfoundry-architecture.md) for the full CF architecture reference.

---

## Developer Experience

### Two Interfaces: CLI and MCP

```
╭──────────────────────────────────────────────────────────────────╮
│                        Developer / AI                             │
╰───────────────┬──────────────────────────────┬───────────────────╯
                │                              │
        ┌───────▽───────┐              ┌───────▽───────┐
        │  mf push      │              │  Claude Code  │
        │  mf logs      │              │  Cursor, etc. │
        │  mf bind      │              │  (AI Tools)   │
        └───────┬───────┘              └───────┬───────┘
                │                              │
        ┌───────▽───────┐              ┌───────▽───────┐
        │  MicroFoundry │              │  MicroFoundry │
        │  CLI (mf)     │              │  MCP Server   │
        └───────┬───────┘              └───────┬───────┘
                │                              │
                ╰──────────────┬───────────────╯
                       ┌───────▽───────┐
                       │  MicroFoundry │
                       │  API Server   │
                       └───────┬───────┘
                               │
                       ┌───────▽───────┐
                       │  Kubernetes   │
                       └───────────────┘
```

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

| MCP Tool | CF Equivalent | Description |
|---|---|---|
| `mf_push` | `cf push` | Deploy an application from source |
| `mf_logs` | `cf logs` | Stream or fetch application logs |
| `mf_bind_service` | `cf bind-service` | Bind a backing service to an app |
| `mf_create_service` | `cf create-service` | Provision a backing service instance |
| `mf_routes` | `cf routes` | List or manage application routes |
| `mf_scale` | `cf scale` | Scale app instances or resources |
| `mf_env` | `cf env` | View or set environment variables |
| `mf_apps` | `cf apps` | List deployed applications |
| `mf_delete` | `cf delete` | Remove a deployed application |

---

## Platform Capabilities

### 1. Application Lifecycle

Deploy any application from source with a single command:

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

- **Build strategies**: Dockerfile → Cloud Native Buildpacks (Paketo) → pack CLI (auto-detected)
- **Registry integration**: Optional push to Harbor/Docker Hub with imagePullSecret management
- **Health checks**: HTTP, Port, Process
- **Scaling**: `mf scale hello-world -i 5`
- **Environment variables**: CF manifest.yml compatible

### 2. Service Broker & Catalog

K8s-native service provisioning with built-in catalog:

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

**Service Binding Flow:**

```
mf bind-service hello-world my-db
        │
        ▽
┌───────────────┐     ┌────────────────┐     ┌──────────────────┐
│ Create Secret │────▶│ Mount as env   │────▶│ Inject into Pod  │
│ mf-svc-my-db  │     │ VCAP_SERVICES  │     │ envFrom: secret  │
└───────────────┘     └────────────────┘     └──────────────────┘
```

Credentials are stored in K8s Secrets and injected as `VCAP_SERVICES` environment variable — exactly like CloudFoundry.

### 3. Observability (Loggregator Equivalent)

Netflix Atlas-inspired auto-instrumentation using Grafana Beyla (eBPF). Applications get full RED metrics **without any code changes** — like Netflix Spectator auto-injects into JVM apps, but using eBPF for any language.

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
│  Recording Rules (pre-computed RED):                    │       │
│    microfoundry:http_request_rate:5m                    │       │
│    microfoundry:http_error_rate:5m              ┌───────▽─────┐ │
│    microfoundry:http_latency_p50/p95/p99:5m     │AlertManager │ │
│                                                  └─────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

**RED Metrics** (Rate, Errors, Duration):
- **Rate**: requests per second per app
- **Errors**: 5xx error ratio
- **Duration**: p50, p95, p99 latency percentiles

**Admin Dashboard** integrates metrics and logs in a single view:
- Performance tab: RED stat cards + Grafana panels + correlated logs
- Logs tab: RED metrics banner + historical log query (Loki) + live SSE streaming
- Alerts: `MFAppHighErrorRate`, `MFAppHighLatency`, `MFAppNoTraffic`, `MFBeylaDown`

### 4. Authentication & Authorization

OIDC-based authentication via **Keycloak** with one-command setup:

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
- **Cookie-based sessions** via gorilla/sessions
- **Optional**: Auth can be disabled entirely for development

### 5. Platform Configuration

The admin dashboard provides UI-based platform configuration:

```
┌─────────────────────────────────────────────────────────────┐
│ Admin UI Sidebar                                             │
│                                                              │
│ OPERATIONS               SETTINGS                           │
│  Dashboard               Clusters                           │
│  Applications            Service Catalog                    │
│  Services                Registry                           │
│  Secrets                 Webhooks                           │
│  Metrics & Alerts        SMTP                               │
│                          Users & Orgs                       │
│                          Platform                           │
└─────────────────────────────────────────────────────────────┘
```

**Settings stored in Kubernetes** (ConfigMap + Secret):
- **Container Registry** — Harbor/Docker Hub URL, credentials, push toggle
- **Webhooks** — HTTP event notifications (app.deployed, app.crashed, alert.fired, etc.)
- **SMTP** — Email notification configuration

### 6. Network & Routing

- **Ingress Controller**: Pluggable (Kong, Nginx, CSP-native API gateways)
- **App routing**: `<app-name>.<domain>` (subdomain per app)
- **Local dev**: `cf-local.dev` with automatic `/etc/hosts` management
- **Cloud**: Real FQDN with DNS and TLS when deploying to EKS/GKE/AKS
- **Service networking**: K8s ClusterIP services for backing services, credentials injected via VCAP_SERVICES

### 7. Secret Management

```bash
$ mf secrets                           # List all secrets
$ mf create-secret api-keys key=val    # Create user secret
$ mf secret my-db                      # Show service credentials
$ mf secret my-db --reveal             # Reveal actual values
```

Two secret types:
- **Service secrets** (`mf-svc-*`): Auto-created when provisioning backing services
- **User secrets** (`mf-secret-*`): Developer-managed key-value pairs

### 8. AI-Native Platform (MCP + Agent)

MicroFoundry exposes itself as a [Model Context Protocol](https://modelcontextprotocol.io/) server, enabling AI tools to be first-class platform operators. AI assistants can push code, bind services, check logs, scale instances, and manage routes — all without leaving the AI-powered workflow.

---

## Admin Web Dashboard

The built-in admin dashboard (`mf admin`, default `:8080`) provides:

- **Application management**: Deploy, scale, delete, view instances with 6-tab detail view (Overview, Instances, Config, Services, Routes, Logs/Performance)
- **Service catalog browser**: Browse and provision services by category with plan visibility control
- **Real-time metrics**: RED metrics per app with Grafana panel integration
- **Log viewer**: Historical query (Loki) + live SSE streaming
- **Multi-cluster management**: Add, remove, switch clusters with health monitoring
- **Secret management**: View and manage secrets (with reveal toggle)
- **Platform settings**: Registry, Webhooks, SMTP configuration via UI
- **Authentication**: Keycloak OIDC login with org management
- **Topology visualization**: Terraform topology editor for service plans
- **40+ API endpoints**: Full JSON API for programmatic access

See [docs/admin-guide.md](docs/admin-guide.md) for the complete dashboard guide.

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
│   ├── admin.go               #   mf admin (web dashboard + auth init)
│   └── setup.go               #   mf setup keycloak / keycloak-realm / keycloak-idp
│
├── pkg/                       # Go packages
│   ├── admin/                 #   Web dashboard + API handlers (60+ routes)
│   │   ├── server.go          #     HTTP server, route registration, auth middleware
│   │   ├── handlers.go        #     App detail, dashboard, tab routing
│   │   ├── api.go             #     JSON API endpoints
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
│   │       ├── templates/     #       18+ page templates
│   │       │   ├── partials/  #       Shared partials (nav, etc.)
│   │       │   └── tabs/      #       HTMX tab partials (8 tabs)
│   │       └── css/           #       Tailwind CSS
│   ├── auth/                  #   OIDC + Keycloak + sessions + orgs
│   │   ├── oidc.go            #     Authorization code flow with PKCE
│   │   ├── session.go         #     Cookie-based session management
│   │   ├── keycloak.go        #     Realm/client/IdP configuration
│   │   ├── org.go             #     Organization & member management
│   │   └── middleware.go      #     InjectUser middleware
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
│   │   ├── app.go             #     App, AppDetail, AppListItem, InstanceStatus
│   │   ├── service.go         #     ServiceType, ServicePlan, ServiceInstance
│   │   ├── cluster.go         #     ClusterInfo, ClusterDetail, NodeInfo
│   │   └── settings.go        #     RegistryConfig, WebhookConfig, SMTPConfig
│   ├── monitoring/            #   Observability stack integration
│   │   ├── prometheus.go      #     Prometheus query client (RED metrics)
│   │   ├── grafana.go         #     Grafana dashboard URL builder
│   │   ├── loki.go            #     Log aggregation client
│   │   ├── alertmanager.go    #     Alert management client
│   │   ├── metrics.go         #     Custom Prometheus metrics
│   │   ├── collector.go       #     Background metrics collection (30s interval)
│   │   └── middleware.go      #     HTTP metrics middleware
│   ├── secrets/               #   K8s Secret management
│   ├── service/               #   Service broker + catalog + provisioning
│   │   ├── catalog.go         #     10 service types, 3 plans each
│   │   ├── manager.go         #     Lifecycle management
│   │   ├── provisioner.go     #     K8s-native provisioning
│   │   ├── binder.go          #     VCAP_SERVICES injection
│   │   ├── vcap.go            #     VCAP_SERVICES formatting
│   │   └── visibility.go      #     Plan visibility toggle (ConfigMap-backed)
│   ├── settings/              #   Platform settings (ConfigMap/Secret store)
│   └── terraform/             #   Terraform topology management
│
├── deploy/
│   ├── k8s/                   # Kubernetes manifests
│   │   ├── base/              #   Base manifests (namespace)
│   │   └── overlays/          #   Kustomize overlays (local, EKS, GKE, AKS)
│   └── monitoring/            # Observability stack
│       ├── install.sh         #   One-command monitoring setup (6 steps)
│       ├── beyla-config.yaml  #   Beyla eBPF DaemonSet + RBAC + NetworkPolicy
│       ├── prometheus-recording-rules.yaml  # RED recording rules
│       ├── kube-prometheus-values.yaml      # Prometheus Helm values
│       ├── loki-values.yaml                 # Loki + Promtail config
│       ├── dashboards/        #   Grafana dashboards (overview, cluster, app-detail)
│       └── alerts/            #   Prometheus alerting rules
│
├── configs/mf.example.yaml   # Example configuration
├── docs/                      # Documentation
│   ├── user-manual.md         #   Complete user guide
│   ├── architecture.md        #   Technical architecture
│   ├── admin-guide.md         #   Admin dashboard guide
│   ├── cloudfoundry-architecture.md  # CF reference architecture
│   └── observability-capacity.md     # Monitoring stack docs
├── Makefile                   # Build targets
├── Dockerfile                 # Container build
├── CLAUDE.md                  # Agent workflow rules + branch strategy
└── LICENSE
```

---

## Tech Stack

| Layer | Technology | Purpose |
|---|---|---|
| **Language** | Go 1.25 | API server, CLI, MCP server, all controllers |
| **CLI Framework** | Cobra + Viper | Command parsing + configuration |
| **Runtime** | Kubernetes | Application scheduling and orchestration |
| **Build** | Cloud Native Buildpacks (Paketo) | Source-to-container builds |
| **Ingress** | Kong / Nginx | API gateway and routing |
| **Metrics** | Prometheus + Grafana | Metrics collection and visualization |
| **Logs** | Promtail + Loki | Log aggregation and querying |
| **Auto-Instrumentation** | Grafana Beyla (eBPF) | Zero-code HTTP metrics |
| **Alerting** | AlertManager | Alert routing and notification |
| **Authentication** | Keycloak + coreos/go-oidc | OIDC authorization code flow |
| **Sessions** | gorilla/sessions | Cookie-based session management |
| **UI** | Go templates + HTMX + Tailwind CSS | Server-side rendering, no JS build step |
| **IaC** | Terraform | Cloud resource provisioning |
| **AI Integration** | Model Context Protocol (MCP) | AI tool platform access |
| **K8s Client** | client-go | Kubernetes API interactions |

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

Copy and edit the example config:

```bash
cp configs/mf.example.yaml configs/mf.yaml
# or for user-wide:
cp configs/mf.example.yaml ~/.mf/mf.yaml
```

Multi-cluster configuration:

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

## Documentation

| Document | Description |
|----------|-------------|
| [User Manual](docs/user-manual.md) | Complete guide to deploying and managing applications |
| [Architecture](docs/architecture.md) | Technical architecture and component design |
| [Admin Guide](docs/admin-guide.md) | Admin dashboard pages, API reference |
| [CF Architecture](docs/cloudfoundry-architecture.md) | CloudFoundry reference architecture |
| [Observability](docs/observability-capacity.md) | Monitoring stack documentation |

---

## License

See [LICENSE](LICENSE).
