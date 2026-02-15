# MicroFoundry

A micro CloudFoundry for Kubernetes with AI-powered development agents.

MicroFoundry brings the CloudFoundry developer experience to Kubernetes, supporting:
- **Application Runtime**: Docker Desktop K8s, EKS, GKE, AKS, native K8s
- **Backing Services**: AWS RDS, Google BigQuery, and more via service-bind
- **Network Services**: AWS API Gateway, Kong, Nginx, and other API gateways
- **Built with**: Go and Rust

## Agent Pipeline

MicroFoundry includes an AI-powered development pipeline (`mf` CLI) that automates Epic-to-PR workflows using 5 specialized agents:

| Agent | Role | Output |
|-------|------|--------|
| **Analyzer** | Analyzes CF codebase, creates detailed implementation plan | GitHub Issue |
| **Data Engineer** | Recommends data structures, schemas, API types | Issue Comment |
| **Product Designer** | Assesses UI needs, generates HTML mockups | Issue Comment |
| **Developer** | Writes code, builds, deploys to K8s, tests | Pull Request |
| **Reviewer** | License, security, performance, cost, sizing checks | PR Review |

## Quick Start

### Prerequisites
- Go 1.23+
- Docker Desktop with Kubernetes enabled
- GitHub CLI (`gh`) authenticated
- Anthropic API key

### Build
```bash
make build
```

### Configure
```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY=sk-ant-...

# Or use the config command
./bin/mf config set anthropic.api_key sk-ant-...
```

### Run Full Pipeline
```bash
mf epic "Implement service binding for AWS RDS" \
    --description "Add service-bind mechanism for AWS RDS backing services"
```

### Run Individual Agents
```bash
# Analyzer only
mf analyze "Implement RDS service binding" -d "..."

# Data Engineer on existing issue
mf data-engineer --issue 42

# Designer on existing issue
mf design --issue 42

# Developer on existing issue
mf develop --issue 42 --branch epic/implement-rds-service-binding

# Reviewer on existing PR
mf review --pr 43
```

## Project Structure

```
microfoundry/
├── cmd/mf/              # CLI entry point (Cobra)
├── pkg/
│   ├── agents/          # Agent implementations
│   │   ├── analyzer/    # Epic analysis → GitHub Issue
│   │   ├── dataengineer/# Data structure recommendations
│   │   ├── designer/    # UI assessment & HTML mockups
│   │   ├── developer/   # Code → Build → Deploy → PR
│   │   └── reviewer/    # License/Security/Perf review
│   ├── claude/          # Claude API client (anthropic-sdk-go)
│   ├── github/          # GitHub operations (gh CLI wrapper)
│   ├── k8s/             # Kubernetes deploy helpers
│   ├── config/          # Configuration management (Viper)
│   └── codebase/        # CF reference repo scanner
├── configs/             # Configuration files
├── deploy/k8s/          # Kubernetes manifests
├── Makefile
└── Dockerfile
```

## Configuration

Configuration is loaded from (highest to lowest precedence):
1. CLI flags
2. Environment variables (`MF_*`, `ANTHROPIC_API_KEY`)
3. `~/.mf/config.yaml`
4. `./configs/mf.yaml`

See [configs/mf.example.yaml](configs/mf.example.yaml) for all options.

## Workflow

1. User runs `mf epic "Feature Title" -d "Description"`
2. **Analyzer** scans CF reference repos, uses Claude with extended thinking, creates GitHub Issue + branch
3. **Data Engineer** reads the Issue, recommends data structures, posts as comment
4. **Product Designer** assesses UI needs, generates HTML mockups if needed
5. **Developer** reads all context, writes code via Claude tool-calling loop, builds, deploys, creates PR
6. **Reviewer** checks license compliance, security, performance, cost, sizing; posts review
7. Human reviews and merges via squash merge

## License

See [LICENSE](LICENSE).
