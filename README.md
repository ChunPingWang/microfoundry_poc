# MicroFoundry

A micro CloudFoundry for Kubernetes — same developer experience, CSP-native infrastructure.

MicroFoundry preserves the CloudFoundry developer and user experience (`cf push`, service binding, route management) while replacing the heavyweight BOSH/Diego infrastructure with modern, cloud-native equivalents. The goal is **simplicity**: instead of operating dozens of CF components, MicroFoundry delegates to managed cloud services and Kubernetes primitives wherever possible.

## What You Can Deploy

- **Web services** — traditional HTTP applications
- **MCP servers** — Model Context Protocol servers for AI tool integration
- **AI Agent workloads** — autonomous agent runtimes and orchestration

All deployable from your development directory with a single `cf push`.

## Design Principles

1. **CF UX, cloud-native infra** — Keep `cf push`, `cf bind-service`, `cf logs` as the developer interface. Replace the underlying machinery with CSP-managed services and Kubernetes.
2. **No custom infrastructure where managed services exist** — App parameters go to AWS Secrets Manager / GCP Secret Manager / Azure Key Vault instead of a custom config server. Logs flow through lightweight collectors to cloud-native backends instead of Loggregator.
3. **Multi-cloud, K8s-only runtime** — Target EKS, GKE, AKS, ECS, and on-premise Kubernetes. No VM-based deployment.
4. **API Gateway as the routing layer** — Replace Gorouter with pluggable API gateways (Kong, Nginx, AWS API Gateway) for endpoint access, rate limiting, and authentication.

## Architecture

| CF Component | MicroFoundry Equivalent | Rationale |
|---|---|---|
| Diego Cell | Kubernetes Pod | K8s is the universal container orchestrator |
| Gorouter | API Gateway (Kong / Nginx / AWS API GW) | Pluggable, CSP-native routing with richer features |
| Cloud Controller | MicroFoundry API Server (Go) | Lightweight API that talks to K8s API directly |
| UAA | K8s RBAC / Dex / Keycloak | Leverage existing identity providers |
| Config Server / Params | CSP Secret Manager (AWS SM, GCP SM, Azure KV) | No custom parameter store needed |
| Service Broker | OSBAPI-compatible broker | Same service binding model as CF |
| Blobstore | S3 / GCS / MinIO | Managed object storage |
| Loggregator | Fluent Bit + Loki (or CloudWatch/Cloud Logging) | Lightweight log collection, CSP-native backends |
| Metrics (Doppler) | Prometheus + Grafana | Industry-standard observability stack |
| Buildpacks | Cloud Native Buildpacks (CNB/Paketo) | Source-to-container without Dockerfile |

See [docs/cloudfoundry-architecture.md](docs/cloudfoundry-architecture.md) for the full CF architecture reference.

## Local Development

For local development, MicroFoundry runs on **Docker Desktop Kubernetes** with the domain `cf-local.dev`:

```
┌─────────────────────────────────────────────────┐
│  Docker Desktop Kubernetes                      │
│                                                 │
│  ┌──────────────┐    ┌───────────────────────┐  │
│  │ Ingress      │    │ myapp.cf-local.dev    │  │
│  │ Controller   │───▶│ (K8s Deployment)      │  │
│  │ (Kong/Nginx) │    └───────────────────────┘  │
│  │              │    ┌───────────────────────┐  │
│  │ *.cf-local   │    │ mcp-server.cf-local   │  │
│  │   .dev       │───▶│   .dev                │  │
│  └──────────────┘    └───────────────────────┘  │
└─────────────────────────────────────────────────┘
```

- **Base domain**: `cf-local.dev`
- **App routing**: `<app-name>.cf-local.dev` (subdomain per app)
- **Host resolution**: MicroFoundry updates the local `hosts` file so subdomains resolve to `127.0.0.1`
- **No FQDN required** — everything runs locally with no external DNS dependency

When deploying to cloud (EKS/GKE/AKS), the domain switches to a real FQDN with proper DNS and TLS.

## Tech Stack

- **Go** — API server, CLI, orchestration, K8s controllers
- **Rust** — Performance-critical runtime components (future)
- **Kubernetes** — Application runtime and orchestration
- **Open Service Broker API** — Backing service integration
- **Prometheus + Grafana** — Metrics and monitoring
- **Fluent Bit + Loki** — Log collection and aggregation
- **Cloud Native Buildpacks** — Source-to-container builds

## Project Structure

```
microfoundry/
├── cmd/                    # CLI entry points
├── pkg/                    # Go packages
│   ├── config/             # Configuration management
│   ├── github/             # GitHub integration
│   └── k8s/                # Kubernetes helpers
├── deploy/k8s/             # Kubernetes manifests
│   ├── base/               # Base manifests
│   └── overlays/           # Kustomize overlays (local, EKS, GKE, AKS)
├── docs/                   # Architecture documentation
├── configs/                # Configuration files
├── Makefile
├── Dockerfile
└── LICENSE
```

## Development

### Prerequisites

- Go 1.23+
- Docker Desktop with Kubernetes enabled
- GitHub CLI (`gh`) authenticated
- kubectl

### Build

```bash
make build
```

## License

See [LICENSE](LICENSE).
