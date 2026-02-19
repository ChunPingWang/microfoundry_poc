# Epic: Observability Hardening — Security, Resilience, Performance & Operability

## Context

PR #20 introduced Netflix Atlas-inspired auto-instrumentation with Grafana Beyla (eBPF), Prometheus recording rules, Loki log integration, and an enhanced admin UI. Seven agent reviews (Developer, Reviewer, DevOps, Security, License, Cost, Performance) identified **28 actionable findings** across 6 themes. This Epic tracks the work to address all findings and harden the observability stack for production readiness.

**Source**: [PR #20 Agent Reviews](https://github.com/younjinjeong/microfoundry/pull/20)
**Depends on**: Epic #18 (Beyla Auto-Instrumentation)

---

## Phase 1: Security Hardening (Critical)

> Sources: Security Agent, DevOps Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 1.1 | **Replace `privileged: true` with minimal capabilities** — Use `CAP_BPF`, `CAP_PERFMON`, `CAP_NET_ADMIN`, `CAP_SYS_PTRACE` instead of full privileged mode. Reduces blast radius if Beyla container is compromised. | P0 | Security |
| 1.2 | **Pin Beyla image to digest** — Change `grafana/beyla:latest` to a pinned version with SHA256 digest (e.g., `grafana/beyla:1.8.5@sha256:...`). Prevents supply chain attacks via mutable tags. | P0 | Security |
| 1.3 | **Sanitize app name for PromQL injection** — Validate app name parameter (alphanumeric + hyphens only) before interpolating into PromQL templates. An app named `foo} or vector(1) #` could alter query semantics. | P0 | Security |
| 1.4 | **Add NetworkPolicy for Beyla DaemonSet** — Restrict Beyla egress to Prometheus only, deny all ingress except Prometheus scraping. Prevents lateral movement from compromised Beyla pod. | P1 | Security |
| 1.5 | **Verify RBAC is read-only** — Audit that ClusterRole grants only `get/list/watch` (no `create/update/delete/patch`). Verify binding uses dedicated ServiceAccount, not `default`. | P1 | Security |

### Solution Plan

**beyla-config.yaml — Replace privileged: true**
```yaml
securityContext:
  privileged: false
  capabilities:
    add: ["BPF", "PERFMON", "NET_ADMIN", "SYS_PTRACE"]
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

**Image pinning**
```yaml
image: grafana/beyla:1.8.5@sha256:<digest>
```

**NetworkPolicy (new file: beyla-networkpolicy.yaml)**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: beyla-network-policy
  namespace: monitoring
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: beyla
  policyTypes: ["Ingress", "Egress"]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring
      ports:
        - port: 9090
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring
```

**PromQL injection prevention (pkg/monitoring/prometheus.go)**
```go
var validAppName = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)

func (p *PrometheusClient) GetAppREDMetrics(ctx context.Context, appName string) (*REDMetrics, error) {
    if !validAppName.MatchString(appName) {
        return nil, fmt.Errorf("invalid app name: %q", appName)
    }
    // ... existing logic
}
```

---

## Phase 2: Operational Resilience (High)

> Sources: Reviewer Agent, Developer Agent, DevOps Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 2.1 | **Add Beyla heartbeat alert** — Alert when `up{job="beyla-metrics"} == 0` for >2 minutes. Beyla failure silently drops all HTTP metrics and alerts, creating a blind spot. | P0 | Reviewer |
| 2.2 | **Circuit breaker for Prometheus queries** — Add timeout and graceful degradation when Prometheus is unreachable. Prevent the observability endpoint from hanging. | P1 | Developer |
| 2.3 | **Add `warnings` field to API responses** — When Prometheus returns partial data (some series but not others), signal incompleteness so the frontend can show a degraded-data indicator. | P1 | Developer |
| 2.4 | **Handle empty `http_route` in recording rules** — Filter or normalize catch-all routes (`/`, `/*`) to avoid noisy aggregation. | P2 | Developer |
| 2.5 | **Document recording rule staleness** — The 5-minute evaluation window introduces up to 5 minutes of staleness in RED metrics. Document this trade-off for operators. | P2 | Reviewer |

### Solution Plan

**Beyla heartbeat alert (microfoundry-alerts.yaml)**
```yaml
- alert: MFBeylaDown
  expr: up{job="beyla-metrics"} == 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Beyla eBPF instrumentation is down"
    description: "Beyla DaemonSet is not being scraped. All HTTP metrics and eBPF-based alerts are unavailable."
```

**Warnings field (pkg/monitoring/prometheus.go)**
```go
type REDMetrics struct {
    RequestRate float64  `json:"request_rate"`
    ErrorRate   float64  `json:"error_rate"`
    LatencyP50  float64  `json:"latency_p50"`
    LatencyP95  float64  `json:"latency_p95"`
    LatencyP99  float64  `json:"latency_p99"`
    HasData     bool     `json:"has_data"`
    Warnings    []string `json:"warnings,omitempty"`
}
```

---

## Phase 3: Performance Optimization (Medium)

> Sources: Performance Agent, Cost Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 3.1 | **Parallelize Prometheus queries with `errgroup`** — The observability endpoint runs 5 queries sequentially. Use `errgroup` to run them concurrently, reducing latency from `5 * query_time` to `max(query_time)`. | P1 | Performance |
| 3.2 | **Profile and right-size Beyla memory** — Actual usage may be 80-150Mi for HTTP-only workloads. Right-sizing from 512Mi to 256Mi halves per-node memory cost. | P1 | Cost |
| 3.3 | **Add CPU limits to Beyla DaemonSet** — Currently no CPU limit. Add 200m request / 500m limit to prevent runaway consumption during traffic spikes. | P2 | Cost |
| 3.4 | **Disable `BEYLA_BPF_TRACK_REQUEST_HEADERS`** — Header tracking significantly increases per-request eBPF map entry size. Disable unless needed for header-based routing. | P2 | Performance |
| 3.5 | **Create dedicated recording rules for alerting** — Alert rules evaluate against raw metrics, carrying full query cost. Create lightweight recording rules specifically for alert evaluation. | P3 | Performance |

### Solution Plan

**Parallel queries (pkg/monitoring/prometheus.go)**
```go
func (p *PrometheusClient) GetAppREDMetrics(ctx context.Context, appName string) (*REDMetrics, error) {
    metrics := &REDMetrics{}
    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error { v, _ := p.instantQuery(ctx, rateQuery); metrics.RequestRate = v; return nil })
    g.Go(func() error { v, _ := p.instantQuery(ctx, errorQuery); metrics.ErrorRate = v; return nil })
    g.Go(func() error { v, _ := p.instantQuery(ctx, p50Query); metrics.LatencyP50 = v; return nil })
    g.Go(func() error { v, _ := p.instantQuery(ctx, p95Query); metrics.LatencyP95 = v; return nil })
    g.Go(func() error { v, _ := p.instantQuery(ctx, p99Query); metrics.LatencyP99 = v; return nil })

    g.Wait()
    return metrics, nil
}
```

---

## Phase 4: Helm Configurability & Multi-Environment (Medium)

> Sources: DevOps Agent, Cost Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 4.1 | **Expose Beyla as Helm values** — `beyla.enabled`, `beyla.resources.limits.memory`, `beyla.targetNamespace`, `beyla.image.tag`. Enables per-environment tuning without template modification. | P1 | DevOps |
| 4.2 | **Add nodeSelector/affinity for Beyla** — Avoid scheduling on control-plane nodes. Add `node-role.kubernetes.io/worker` selector or toleration logic. | P1 | DevOps |
| 4.3 | **Make Promtail paths configurable** — Support containerd-based clusters (EKS 1.24+, GKE) where log paths differ (`/var/log/containers` symlinks). | P2 | DevOps |
| 4.4 | **Conditional Grafana URL for dev vs prod** — HSTS fix (`localhost:3000`) should not affect production where TLS is enabled. Use environment-based conditional. | P2 | DevOps |

### Solution Plan

**Beyla Helm values (beyla-values.yaml)**
```yaml
beyla:
  enabled: true
  image:
    repository: grafana/beyla
    tag: "1.8.5"
    digest: "sha256:..."
  resources:
    requests:
      memory: 128Mi
      cpu: 100m
    limits:
      memory: 512Mi
      cpu: 500m
  targetNamespaces:
    - microfoundry
  nodeSelector:
    node-role.kubernetes.io/worker: ""
```

---

## Phase 5: Code Quality & Testing (Low)

> Sources: Developer Agent, Reviewer Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 5.1 | **Unit tests for YAML template rendering** — Render Beyla DaemonSet template with sample values and validate against YAML parser to catch whitespace issues. | P2 | Developer |
| 5.2 | **Dashboard-as-code with Grafonnet** — 972+ lines of Grafana JSON is fragile. Generate from Grafonnet (jsonnet) for maintainability. | P3 | Developer |
| 5.3 | **Verify tab deep-linking** — Ensure `/apps/foo?tab=performance` works correctly for incident response workflows. Test tab state preservation across navigation. | P2 | Developer |
| 5.4 | **BFF pattern evaluation** — Evaluate separating `/observability` into `/metrics` and `/logs` sub-resources to decouple log and metric availability. | P3 | Reviewer |

---

## Phase 6: Capacity Planning Documentation (Low)

> Sources: Cost Agent, DevOps Agent, Performance Agent

| # | Task | Priority | Source |
|---|------|----------|--------|
| 6.1 | **Document cardinality growth** — `num_apps * num_routes * num_status_codes` new series per Beyla. Estimate for 20, 50, 100 apps. | P2 | DevOps, Cost |
| 6.2 | **Document storage projections** — ~300MB/month for 20 apps at 15s scrape. Scale linearly. | P2 | Cost |
| 6.3 | **Document scaling breakpoints** — 50+ nodes: add namespace filtering; 100+ apps: consider Thanos/federation. | P3 | Cost |
| 6.4 | **Document eBPF overhead** — <3us per request, <1% throughput impact at 10K req/s. Cite Grafana benchmarks. | P3 | Performance |

---

## File Summary (Estimated)

| Phase | Files | Action |
|-------|-------|--------|
| 1. Security | `deploy/monitoring/beyla-config.yaml`, `deploy/monitoring/beyla-networkpolicy.yaml` (NEW), `pkg/monitoring/prometheus.go` | 2 modify, 1 new |
| 2. Resilience | `deploy/monitoring/alerts/microfoundry-alerts.yaml`, `pkg/monitoring/prometheus.go`, `pkg/admin/performance_handlers.go` | 3 modify |
| 3. Performance | `pkg/monitoring/prometheus.go`, `deploy/monitoring/beyla-config.yaml`, `deploy/monitoring/prometheus-recording-rules.yaml` | 3 modify |
| 4. Helm | `deploy/monitoring/beyla-values.yaml` (NEW), `deploy/monitoring/install.sh`, `deploy/monitoring/loki-values.yaml` | 1 new, 2 modify |
| 5. Quality | Tests (NEW), `pkg/admin/performance_handlers.go` | 1-2 new, 1 modify |
| 6. Docs | `docs/observability-capacity.md` (NEW) | 1 new |

**Total: ~4 new files, ~8 modified files across 6 phases**

## Priority Summary

| Priority | Count | Theme |
|----------|-------|-------|
| P0 (Critical) | 4 | Security (capabilities, image pinning, PromQL injection), Beyla heartbeat |
| P1 (High) | 7 | NetworkPolicy, circuit breaker, parallel queries, Helm values, node affinity |
| P2 (Medium) | 8 | CPU limits, Promtail paths, testing, deep-linking, documentation |
| P3 (Low) | 5 | Grafonnet, BFF evaluation, scaling docs, alert recording rules |

## Acceptance Criteria

- [ ] Beyla runs with minimal capabilities (no `privileged: true`)
- [ ] Beyla image pinned to specific digest
- [ ] PromQL injection prevented via app name validation
- [ ] NetworkPolicy restricts Beyla network access
- [ ] `MFBeylaDown` alert fires when Beyla is unavailable
- [ ] Prometheus queries parallelized in observability endpoint
- [ ] Beyla configurable via Helm values
- [ ] Capacity planning document published
