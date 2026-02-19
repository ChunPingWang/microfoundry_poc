## Analyzer Agent Report

**Agent**: Analyzer | **Label**: `analysis`

---

### Current State Assessment

MicroFoundry uses a **hardcoded single-cluster architecture**:
- 5 out of 6 CLI commands hardcode `docker-desktop` context (only `admin` reads from config)
- Admin server holds a single `*k8s.Client` instance
- Config supports only `kubernetes.context` + `kubernetes.namespace`
- `k8s.NewClient(context, namespace, domain)` already supports any kubeconfig context — the client-go library handles multi-cluster natively

### Key Finding: Zero New Dependencies Needed

The existing `k8s.io/client-go` already supports multi-context kubeconfig. We only need:
1. A **ClientManager** to pool/cache clients per cluster
2. Updated **config schema** for multiple clusters
3. Updated **admin server** to reference ClientManager instead of single Client
4. **Cluster management UI** pages and handlers

### Architecture Recommendation

```
Config (mf.yaml)           ClientManager (pkg/k8s/manager.go)
┌─────────────────┐       ┌──────────────────────────────────┐
│ kubernetes:      │       │ clients: map[string]*Client      │
│   clusters:      │──────>│ config:  map[string]ClusterConfig│
│     docker-desk: │       │ active:  string                  │
│     eks-prod:    │       │ mu:      sync.RWMutex            │
│     gke-staging: │       │                                  │
│   active: docker │       │ GetClient(id) → *Client          │
└─────────────────┘       │ SetActive(id)                    │
                           │ ListClusters() → []ClusterInfo   │
                           │ AddCluster(cfg)                  │
                           │ RemoveCluster(id)                │
                           └──────────────────────────────────┘
                                        │
                           ┌────────────┼────────────┐
                           ▼            ▼            ▼
                       Client(DD)   Client(EKS)  Client(GKE)
```

### Cluster Selection Strategy

**Priority order** (for both admin UI and CLI):
1. Explicit flag (`--cluster eks-prod`)
2. URL query param (`?cluster=eks-prod`) / Cookie (admin UI)
3. Config default (`kubernetes.active`)

### Risk Areas

| Risk | Mitigation |
|------|-----------|
| Breaking CLI change (hardcoded context removed) | Backwards-compatible config migration |
| Stale kubeconfig credentials | Health check endpoint, connection status display |
| Client pool memory | LRU eviction for unused clients |
| Namespace conflicts across clusters | Per-cluster namespace config |

### Impact: 5 new files, 13 modified files, zero new Go dependencies
