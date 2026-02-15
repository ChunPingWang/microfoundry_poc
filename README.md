# MicroFoundry

A micro CloudFoundry for Kubernetes.

MicroFoundry brings the CloudFoundry developer experience — `cf push`, service binding, route management — to Kubernetes, without the complexity of a full CF deployment.

## Features

- **Application Runtime**: Deploy apps to Kubernetes (Docker Desktop, EKS, GKE, AKS, native K8s)
- **Service Binding**: Bind backing services (AWS RDS, Google BigQuery, etc.) via Open Service Broker API
- **Network Services**: Integrate API gateways (Kong, Nginx, AWS API Gateway) as routing layer
- **Multi-Tenancy**: Organizations, spaces, and roles mapped to K8s namespaces and RBAC
- **Buildpack Support**: Cloud Native Buildpacks (CNB/Paketo) for source-to-container

## Architecture

MicroFoundry replaces CloudFoundry's BOSH/Diego infrastructure with Kubernetes-native equivalents:

| CF Component | MicroFoundry Equivalent |
|-------------|------------------------|
| Diego Cell | Kubernetes Pod |
| Gorouter | Ingress Controller (Kong/Nginx) |
| Cloud Controller | MicroFoundry API Server (Go) |
| UAA | K8s RBAC / Dex / Keycloak |
| Service Broker | OSBAPI-compatible broker |
| Blobstore | S3 / GCS / MinIO |
| Loggregator | Fluentd + Loki / native K8s logging |

See [docs/cloudfoundry-architecture.md](docs/cloudfoundry-architecture.md) for the full CF architecture reference.

## Tech Stack

- **Go** — API server, CLI, orchestration, K8s controllers
- **Rust** — Performance-critical runtime components (future)
- **Kubernetes** — Application runtime and orchestration
- **Open Service Broker API** — Backing service integration

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
└── CLAUDE.md               # Development agent workflow
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

### Development Workflow

This project uses **Claude Code development agents** defined in [CLAUDE.md](CLAUDE.md). When an Epic is requested, Claude executes 5 sequential agents:

1. **Analyzer** — Creates GitHub Issue with detailed implementation plan
2. **Data Engineer** — Posts data structure recommendations on the Issue
3. **Product Designer** — Assesses UI needs, posts mockups if needed
4. **Developer** — Writes code, builds, tests, creates Pull Request
5. **Reviewer** — Reviews PR for license, security, performance, cost

All work is tracked through GitHub Issues and PRs with squash merge to `main`.

## License

See [LICENSE](LICENSE).
