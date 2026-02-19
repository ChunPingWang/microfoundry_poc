## Product Designer Agent Report

**Agent**: Product Designer | **Label**: `design`

---

### UI Design: Kubernetes Clusters Management

#### Navigation Update

Add "Kubernetes Clusters" menu item between "Applications" and "Services" in the sidebar:

```
Navigation:
├── Dashboard
├── Applications
├── Kubernetes Clusters  ← NEW (with server icon)
├── Services
├── Secrets
├── Users & Orgs
└── Configuration
```

#### Global Cluster Selector (Header)

Add a cluster dropdown to the layout header area, visible on ALL pages:

```
+------------------------------------------------------------------+
| MicroFoundry Admin                                                |
|                                                                    |
| Dashboard  Applications  Clusters  ...                             |
+------------------------------------------------------------------+
|                              Cluster: [Docker Desktop (Local) ▼]  |
|                                       ├─ Docker Desktop (Local) ✓ |
|                                       ├─ AWS EKS Production       |
|                                       ├─ GCP GKE Staging          |
|                                       └─ Manage Clusters...       |
+------------------------------------------------------------------+
```

**Cluster dropdown**: In the header bar, right side. Shows active cluster with a green dot indicator. Selecting a different cluster sets a cookie and reloads the current page.

#### Cluster List Page (`/clusters`)

```
+------------------------------------------------------------------+
| Kubernetes Clusters                              [+ Add Cluster]  |
+------------------------------------------------------------------+
| Name              | Provider     | Status    | Nodes | Apps | NS         | Actions         |
+-------------------+--------------+-----------+-------+------+------------+-----------------+
| Docker Desktop    | docker-desk  | Connected | 1     | 1    | microfndry | Active ✓  | Del |
| AWS EKS Prod      | eks          | Connected | 3     | 12   | microfndry | Set Active| Del |
| GCP GKE Staging   | gke          | Error     | -     | -    | microfndry | Set Active| Del |
+-------------------+--------------+-----------+-------+------+------------+-----------------+
```

**Provider Badges**:

| Provider | Badge Color |
|----------|-------------|
| docker-desktop | `bg-gray-100 text-gray-800` |
| eks | `bg-orange-100 text-orange-800` |
| gke | `bg-blue-100 text-blue-800` |
| aks | `bg-cyan-100 text-cyan-800` |
| native | `bg-green-100 text-green-800` |

**Status Indicators**:

| Status | Style |
|--------|-------|
| Connected | `bg-green-100 text-green-800` + green dot |
| Disconnected | `bg-red-100 text-red-800` + red dot |
| Error | `bg-red-100 text-red-800` + warning icon |

**Active Cluster**: Highlighted row with `bg-blue-50 border-l-4 border-blue-500`

#### Add Cluster Form

```
+------------------------------------------------------------------+
| Add Kubernetes Cluster                                             |
+------------------------------------------------------------------+
|                                                                    |
|  Display Name     [________________________]                       |
|                                                                    |
|  Provider         [Docker Desktop ▼]                               |
|                    ├─ Docker Desktop                               |
|                    ├─ Amazon EKS                                   |
|                    ├─ Google GKE                                   |
|                    ├─ Azure AKS                                    |
|                    └─ Native Kubernetes                            |
|                                                                    |
|  Kubeconfig Context  [docker-desktop ▼]                            |
|                       (auto-populated from ~/.kube/config)          |
|                                                                    |
|  Namespace        [microfoundry___________]                        |
|                                                                    |
|  App Domain       [cf-local.dev___________]                        |
|                                                                    |
|  Region           [________________________]  (optional)           |
|                                                                    |
|  [Test Connection]                        [Cancel] [Add Cluster]   |
|                                                                    |
+------------------------------------------------------------------+
```

**"Test Connection" button**: HTMX POST to `/api/clusters/test`, shows success/failure inline without page reload.

#### Cluster Detail Page (`/clusters/{id}`)

```
+------------------------------------------------------------------+
| ← Back    Docker Desktop (Local)              [Connected] ●       |
+------------------------------------------------------------------+
|                                                                    |
| Cluster Info                     Resources                         |
| ┌──────────────────────────┐    ┌──────────────────────────┐      |
| │ Provider   docker-desktop│    │ CPU       4/4 cores      │      |
| │ Context    docker-desktop│    │ Memory    3.8G / 7.7G    │      |
| │ Namespace  microfoundry  │    │ Pods      5 / 110        │      |
| │ Domain     cf-local.dev  │    │                          │      |
| │ K8s Version v1.34.1     │    │                          │      |
| └──────────────────────────┘    └──────────────────────────┘      |
|                                                                    |
| Nodes                                                              |
| ┌──────────────────────────────────────────────────────────┐      |
| │ Name            │ Status │ Roles         │ CPU  │ Memory │      |
| │ docker-desktop  │ Ready  │ control-plane │ 4    │ 7.7Gi  │      |
| └──────────────────────────────────────────────────────────┘      |
|                                                                    |
| Applications (1)                                                   |
| ┌──────────────────────────────────────────────────────────┐      |
| │ hello-world  │  started  │  1/1  │  hello-world.cf-...  │      |
| └──────────────────────────────────────────────────────────┘      |
|                                                                    |
|                                        [Edit] [Remove Cluster]     |
+------------------------------------------------------------------+
```

#### UX Patterns

- **Cluster switching**: Dropdown in header, sets `mf-cluster` cookie, page reloads with new cluster context
- **Connection status**: Real-time via HTMX polling (every 30s) on cluster list
- **Test connection**: Inline HTMX feedback on add form — shows checkmark or error message
- **Confirmation**: "Remove Cluster" requires `hx-confirm` dialog
- **Empty state**: No clusters registered → "Add your first cluster" CTA with kubeconfig setup guide
- **Active badge**: Current active cluster shows green "Active" badge in list; switching cluster changes badge
- **Error handling**: Disconnected clusters show warning, disable "Set Active" button, show "Reconnect" action
- **Responsive**: On smaller screens, hide Region and Namespace columns

#### Integration with Existing Pages

1. **Dashboard**: Show "Active Cluster: {name}" card, replace hardcoded context display
2. **Applications**: Show cluster badge on each app if viewing cross-cluster; filter by selected cluster
3. **Header**: All pages show cluster dropdown (part of layout.html)
4. **Config**: Move cluster settings to dedicated Clusters page; Config page links to it
