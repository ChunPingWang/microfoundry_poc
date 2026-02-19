## Product Design Assessment

**Agent**: Product Designer | **Label**: `design`

---

### UI Assessment: Major UI Enhancement Required

This epic redesigns two key pages: the **App List** and **App Detail**. The list gets more columns and filtering. The detail gets a tabbed layout with 6 sections.

### Enhanced App List Design

```
+------------------------------------------------------------------+
| Applications                                      [Filter ▼] [🔄] |
+------------------------------------------------------------------+
| Name        | Org    | Owner  | State   | Inst | Mem  | Type   | Routes              | Created  | Actions     |
+-------------+--------+--------+---------+------+------+--------+---------------------+----------+-------------+
| hello-world | micro  | younj  | started | 1/1  | 128M | docker | hello.cf-local.dev  | 2h ago   | Scale | Del |
| api-server  | micro  | younj  | started | 3/3  | 512M | cnb    | api.cf-local.dev    | 1d ago   | Scale | Del |
| worker      | micro  | admin  | stopped | 0/1  | 256M | docker | -                   | 5d ago   | Scale | Del |
+------------------------------------------------------------------+
```

**New columns**: Org, Owner, Memory, Type (lifecycle), Created
**Filtering**: Dropdown filter by State (All / Started / Stopped)
**Responsive**: On smaller screens, hide Org, Owner, Type columns

### App List Column Design

| Column | Width | Style |
| --- | --- | --- |
| Name | auto | Blue link, font-medium |
| Org | 80px | Gray text, truncate |
| Owner | 80px | Gray text, truncate |
| State | 90px | Color badge (green/red/yellow) |
| Instances | 60px | `running/total` format |
| Memory | 60px | e.g., "128M", "1G" |
| Type | 80px | Pill badge: blue=docker, green=buildpack, purple=cnb |
| Routes | auto | Blue links, comma-separated |
| Created | 80px | Relative time via `timeAgo` |
| Actions | 120px | Scale + Delete buttons |

### Lifecycle Type Badges

| Type | Badge Color |
| --- | --- |
| docker | `bg-blue-100 text-blue-800` |
| buildpack | `bg-green-100 text-green-800` |
| cnb | `bg-purple-100 text-purple-800` |

### App Detail — Tabbed Layout Design

```
+------------------------------------------------------------------+
| ← Back    hello-world                         [started] ●        |
+------------------------------------------------------------------+
| [Overview] [Instances] [Config] [Services] [Routes] [Logs]       |
+------------------------------------------------------------------+
|                                                                    |
|  Tab Content Area (HTMX-swapped partials)                         |
|                                                                    |
+------------------------------------------------------------------+
```

**Tab switching**: HTMX `hx-get` loads tab content as partial, no full page reload.
Active tab: `border-b-2 border-blue-600 text-blue-600`
Inactive tab: `text-gray-500 hover:text-gray-700`

### Tab 1: Overview

```
+------------------------------------------------------------------+
| Overview                                                          |
+------------------------------------------------------------------+
| ┌─────────────────────┐  ┌─────────────────────┐                 |
| │ State     started   │  │ Instances   1/1     │                 |
| │ Owner     younjinj  │  │ Memory      128M    │                 |
| │ Org       microfndy │  │ CPU         100m    │                 |
| │ Created   2h ago    │  │ Disk        1024M   │                 |
| └─────────────────────┘  └─────────────────────┘                 |
|                                                                    |
| Routes                                                            |
| ┌─────────────────────────────────────────────────┐               |
| │ hello-world.cf-local.dev  /  http               │               |
| └─────────────────────────────────────────────────┘               |
|                                                                    |
| Build Info                                                        |
| ┌─────────────────────────────────────────────────┐               |
| │ Type       docker                                │               |
| │ Image      microfoundry/hello-world:latest       │               |
| │ Health     port :8080 (interval 10s)             │               |
| └─────────────────────────────────────────────────┘               |
+------------------------------------------------------------------+
```

### Tab 2: Instances

```
| #  | State    | Since   | Restarts | Node            | Pod Name                    |
|----|----------|---------|----------|-----------------|-----------------------------|
| 0  | RUNNING  | 2h ago  | 0        | docker-desktop  | hello-world-6c4b-j48q4      |
```

Refreshes every 5s via HTMX polling.

### Tab 3: Configuration

```
Environment Variables
┌──────────────────────────────────────────────────┐
│ PORT        8080                                  │
│ DB_PASSWORD ••••••••  [Show]                      │
└──────────────────────────────────────────────────┘

Resource Limits
┌──────────────────────────────────────────────────┐
│ Memory Request  64Mi    │  Memory Limit  128Mi   │
│ CPU Request     100m    │  CPU Limit     -       │
└──────────────────────────────────────────────────┘

Labels
┌──────────────────────────────────────────────────┐
│ app.kubernetes.io/name        hello-world         │
│ app.kubernetes.io/managed-by  microfoundry        │
└──────────────────────────────────────────────────┘
```

Secret env vars (names containing PASSWORD, SECRET, KEY, TOKEN) are masked with `••••••••` and a "Show" toggle.

### Tab 4: Services

```
┌──────────────────────────────────────────────────┐
│  (database icon)  No services bound               │
│                                                    │
│  Use `mf bind-service` to bind a backing service   │
└──────────────────────────────────────────────────┘
```

When services exist:
```
| Service Name | Type     | Status |
|-------------|----------|--------|
| my-postgres | managed  | bound  |
| redis-cache | managed  | bound  |
```

### Tab 5: Routes

```
| Host        | Domain       | Path | Protocol | URL                          |
|------------|--------------|------|----------|------------------------------|
| hello-world | cf-local.dev | /    | http     | http://hello-world.cf-local.dev |
```

### Tab 6: Logs

Existing SSE log streaming panel, moved into tab. Same toggle button + terminal-style output.

### UX Patterns

- **Tab persistence**: URL fragment `#instances`, `#config` etc. so refreshing stays on same tab
- **Env var masking**: Values matching sensitive patterns masked by default, toggleable per-row
- **Mobile**: Tabs become scrollable horizontal strip on small screens
- **Empty states**: Each tab has a helpful empty state with guidance text
- **Loading states**: Show skeleton/spinner while tab content loads via HTMX
