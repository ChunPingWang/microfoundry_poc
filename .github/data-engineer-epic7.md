## Data Engineer Agent Report

**Agent**: Data Engineer | **Label**: `data`

---

### Data Model Recommendations

#### 1. ClusterConfig (Config file representation)

```go
// pkg/config/config.go
type ClusterConfig struct {
    Name      string `mapstructure:"name"      json:"name"`
    Context   string `mapstructure:"context"   json:"context"`    // kubeconfig context name
    Namespace string `mapstructure:"namespace" json:"namespace"`  // MF namespace
    Domain    string `mapstructure:"domain"    json:"domain"`     // App domain (e.g. cf-local.dev)
    Provider  string `mapstructure:"provider"  json:"provider"`   // docker-desktop, eks, gke, aks, native
    Region    string `mapstructure:"region"    json:"region,omitempty"`
    Enabled   bool   `mapstructure:"enabled"   json:"enabled"`
}

type KubernetesConfig struct {
    // New: multi-cluster
    Clusters map[string]ClusterConfig `mapstructure:"clusters" json:"clusters"`
    Active   string                   `mapstructure:"active"   json:"active"`

    // Legacy: single-cluster (for migration)
    Context   string `mapstructure:"context"   json:"-"`
    Namespace string `mapstructure:"namespace" json:"-"`
}
```

#### 2. ClusterInfo (Runtime status, returned by API)

```go
// pkg/models/cluster.go
type ClusterInfo struct {
    ID          string    `json:"id"`             // Config key (e.g. "docker-desktop")
    Name        string    `json:"name"`           // Display name
    Provider    string    `json:"provider"`       // docker-desktop, eks, gke, aks, native
    Region      string    `json:"region,omitempty"`
    Context     string    `json:"context"`        // kubeconfig context
    Namespace   string    `json:"namespace"`
    Domain      string    `json:"domain"`
    Status      string    `json:"status"`         // connected, disconnected, error
    Enabled     bool      `json:"enabled"`
    IsActive    bool      `json:"is_active"`      // Currently selected
    NodeCount   int       `json:"node_count"`
    AppCount    int       `json:"app_count"`
    Version     string    `json:"version"`        // K8s server version
    StatusMsg   string    `json:"status_message,omitempty"`
}
```

#### 3. ClusterDetail (Extended info for detail page)

```go
type ClusterDetail struct {
    ClusterInfo
    Nodes        []NodeInfo        `json:"nodes"`
    ResourceUsage ResourceSummary  `json:"resource_usage"`
    CreatedAt    string            `json:"created_at,omitempty"`
}

type NodeInfo struct {
    Name      string `json:"name"`
    Status    string `json:"status"`    // Ready, NotReady
    Roles     string `json:"roles"`     // control-plane, worker
    Version   string `json:"version"`
    OS        string `json:"os"`        // linux/amd64
    CPU       string `json:"cpu"`       // Allocatable CPU
    Memory    string `json:"memory"`    // Allocatable memory
}

type ResourceSummary struct {
    CPUCapacity    string `json:"cpu_capacity"`
    CPUUsed        string `json:"cpu_used"`
    MemoryCapacity string `json:"memory_capacity"`
    MemoryUsed     string `json:"memory_used"`
    PodCapacity    int    `json:"pod_capacity"`
    PodUsed        int    `json:"pod_used"`
}
```

#### 4. Provider Constants

```go
const (
    ProviderDockerDesktop = "docker-desktop"
    ProviderEKS           = "eks"
    ProviderGKE           = "gke"
    ProviderAKS           = "aks"
    ProviderNative        = "native"
)

var ValidProviders = []string{
    ProviderDockerDesktop, ProviderEKS, ProviderGKE, ProviderAKS, ProviderNative,
}
```

#### 5. Config File Schema

```yaml
kubernetes:
  clusters:
    local:
      name: "Docker Desktop (Local)"
      context: "docker-desktop"
      namespace: "microfoundry"
      domain: "cf-local.dev"
      provider: "docker-desktop"
      enabled: true
    eks-prod:
      name: "AWS EKS Production"
      context: "arn:aws:eks:us-west-2:123456:cluster/prod"
      namespace: "microfoundry"
      domain: "apps.example.com"
      provider: "eks"
      region: "us-west-2"
      enabled: true
    gke-staging:
      name: "GCP GKE Staging"
      context: "gke_project_zone_cluster"
      namespace: "microfoundry"
      domain: "staging.example.com"
      provider: "gke"
      region: "us-central1"
      enabled: true
  active: "local"
```

#### 6. ClientManager Interface

```go
// pkg/k8s/manager.go
type ClientManager struct {
    clients  map[string]*Client           // Lazy-loaded client cache
    configs  map[string]config.ClusterConfig // Cluster configurations
    active   string                       // Currently active cluster ID
    mu       sync.RWMutex
}

// Core methods
func NewClientManager(clusters map[string]config.ClusterConfig, active string) *ClientManager
func (cm *ClientManager) GetClient(clusterID string) (*Client, error)
func (cm *ClientManager) GetActiveClient() (*Client, error)
func (cm *ClientManager) SetActive(clusterID string) error
func (cm *ClientManager) GetActive() string

// CRUD
func (cm *ClientManager) AddCluster(id string, cfg config.ClusterConfig) error
func (cm *ClientManager) RemoveCluster(id string) error
func (cm *ClientManager) ListClusters() []config.ClusterConfig

// Health
func (cm *ClientManager) CheckHealth(clusterID string) (string, error)  // returns status
```

#### 7. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/clusters` | List all registered clusters with status |
| GET | `/api/clusters/{id}` | Cluster detail with nodes/resources |
| POST | `/api/clusters` | Register a new cluster |
| PUT | `/api/clusters/{id}` | Update cluster config |
| DELETE | `/api/clusters/{id}` | Remove cluster |
| POST | `/api/clusters/{id}/activate` | Set as active cluster |
| GET | `/api/clusters/{id}/health` | Health check |

#### 8. Migration Logic

```go
func (cfg *KubernetesConfig) Migrate() {
    if cfg.Clusters != nil {
        return // Already new format
    }
    // Auto-migrate from legacy single-cluster config
    id := cfg.Context
    if id == "" {
        id = "docker-desktop"
    }
    cfg.Clusters = map[string]ClusterConfig{
        id: {
            Name:      id,
            Context:   cfg.Context,
            Namespace: cfg.Namespace,
            Domain:    "cf-local.dev",
            Provider:  detectProvider(cfg.Context),
            Enabled:   true,
        },
    }
    cfg.Active = id
}

func detectProvider(context string) string {
    switch {
    case strings.Contains(context, "docker-desktop"):
        return "docker-desktop"
    case strings.Contains(context, "eks") || strings.Contains(context, "aws"):
        return "eks"
    case strings.Contains(context, "gke") || strings.Contains(context, "gke_"):
        return "gke"
    case strings.Contains(context, "aks") || strings.Contains(context, "azure"):
        return "aks"
    default:
        return "native"
    }
}
```
