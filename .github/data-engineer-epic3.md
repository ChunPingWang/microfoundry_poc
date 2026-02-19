## Data Engineering Review

**Agent**: Data Engineer | **Label**: `data-engineering`

---

### Data Architecture Decision: K8s as Single Source of Truth

No database required. All application metadata is stored as **K8s Deployment annotations** at deploy time and extracted at query time. This eliminates data synchronization issues — the K8s API is always the canonical state.

### New Data Models

#### `AppDetail` — Comprehensive App View (new)

| Field | Type | Source |
| --- | --- | --- |
| Name | string | Deployment.metadata.name |
| GUID | string | Annotation `microfoundry.io/guid` |
| State | string | Derived from running pod count |
| Organization | string | Deployment.metadata.namespace |
| Owner | string | Annotation `microfoundry.io/owner` |
| LifecycleType | string | Annotation `microfoundry.io/lifecycle` |
| Buildpacks | []string | Annotation `microfoundry.io/buildpacks` |
| ImageRef | string | Container spec `.image` |
| ImageDigest | string | Pod `.status.containerStatuses[0].imageID` |
| Command | string | Container spec `.command` |
| Instances | int | Deployment `.spec.replicas` |
| RunningCount | int | Count of Running pods |
| MemoryMB | int | Container `.resources.limits.memory` (parse Mi) |
| DiskMB | int | Annotation `microfoundry.io/disk-mb` |
| CPUMillis | int | Container `.resources.requests.cpu` (parse m) |
| HealthCheck | HealthCheck | Container `.readinessProbe` |
| Routes | []RouteDetail | Ingress rules (host + path) |
| Env | map[string]string | Container `.env[]` |
| Labels | map[string]string | Deployment `.metadata.labels` |
| Annotations | map[string]string | Deployment `.metadata.annotations` |
| Services | []ServiceBindingInfo | Annotation `microfoundry.io/services` |
| Secrets | []SecretInfo | K8s Secrets with matching label |
| InstanceList | []InstanceDetail | Pod list |
| CreatedAt | time.Time | Deployment `.metadata.creationTimestamp` |

#### `InstanceDetail` — Enhanced Instance (extends InstanceStatus)

| Field | Type | Source | NEW |
| --- | --- | --- | --- |
| Index | int | Pod index | existing |
| State | string | ContainerStatus.State | existing |
| Since | time.Time | Running.StartedAt | existing |
| RestartCount | int | ContainerStatus.RestartCount | **NEW** |
| NodeName | string | Pod.Spec.NodeName | **NEW** |
| PodName | string | Pod.Name | **NEW** |
| ImageID | string | ContainerStatus.ImageID | **NEW** |

#### `RouteDetail` — Full Route Info

| Field | Type | Source |
| --- | --- | --- |
| Host | string | IngressRule.Host (split at domain) |
| Domain | string | IngressRule.Host (domain part) |
| Path | string | IngressRule.HTTP.Paths[].Path |
| URL | string | Computed: host.domain/path |
| Protocol | string | "http" (or "https" if TLS configured) |

#### `ServiceBindingInfo` — Service Display (new)

| Field | Type | Source |
| --- | --- | --- |
| Name | string | Annotation parsed from comma-separated list |
| Type | string | "managed" or "user-provided" |
| Status | string | "bound" |

#### `SecretInfo` — Secret Metadata (new, no values exposed)

| Field | Type | Source |
| --- | --- | --- |
| Name | string | Secret.metadata.name |
| Type | string | Secret.type (Opaque, kubernetes.io/tls, etc.) |
| KeyCount | int | len(Secret.data) |
| CreatedAt | string | Secret.metadata.creationTimestamp |

### K8s Annotation Schema for Deploy-Time Metadata

All annotations use the `microfoundry.io/` prefix:

| Annotation Key | Value | Set At |
| --- | --- | --- |
| `microfoundry.io/owner` | Git user or --owner flag | `mf push` |
| `microfoundry.io/org` | Namespace name | `mf push` |
| `microfoundry.io/lifecycle` | docker / buildpack / cnb | `mf push` |
| `microfoundry.io/buildpacks` | Comma-separated names | `mf push` |
| `microfoundry.io/source-image` | Original image ref | `mf push` |
| `microfoundry.io/services` | Comma-separated service names | `mf push` |
| `microfoundry.io/disk-mb` | Disk quota in MB | `mf push` |
| `microfoundry.io/created-at` | RFC3339 timestamp | `mf push` |

### App List View Model (enhanced)

| Column | Field | Description |
| --- | --- | --- |
| Name | Name | App name (link to detail) |
| Org | Organization | K8s namespace |
| Owner | Owner | From annotation |
| State | State | started/stopped badge |
| Instances | RunningCount/TotalCount | Running vs desired |
| Memory | MemoryMB | Container memory limit |
| Type | LifecycleType | docker/buildpack/cnb badge |
| Routes | Routes | Hostname links |
| Created | CreatedAt | Relative time (e.g., "2h ago") |
| Actions | - | Scale / Delete buttons |

### Recommendation

All data models remain **stateless** — no database, no cache. Every query goes directly to the K8s API. The `GetAppDetail()` function reads Deployment spec + annotations + Pod status + Ingress rules in a single handler call, keeping the architecture simple. For secret values, only metadata (name, type, key count) is exposed — never the actual secret data.
