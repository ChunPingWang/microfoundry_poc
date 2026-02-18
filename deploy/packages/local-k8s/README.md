# MicroFoundry -- Local Kubernetes Deployment

Deploy MicroFoundry on a local Kubernetes cluster (Docker Desktop, kind, or minikube).

---

## Prerequisites

| Tool | Minimum Version | Install |
|------|-----------------|---------|
| Docker Desktop (with K8s enabled), kind, or minikube | Docker Desktop 4.x / kind 0.20+ / minikube 1.30+ | [Docker Desktop](https://www.docker.com/products/docker-desktop/) / [kind](https://kind.sigs.k8s.io/) / [minikube](https://minikube.sigs.k8s.io/) |
| kubectl | 1.27+ | https://kubernetes.io/docs/tasks/tools/ |
| Helm | 3.12+ | https://helm.sh/docs/intro/install/ |

Verify that your cluster is running:

```bash
kubectl cluster-info
```

---

## Quick Install

```bash
# Basic install (MicroFoundry only)
bash install.sh

# Install with monitoring stack (Prometheus + Grafana + Loki + Beyla)
bash install.sh --with-monitoring
```

The script will:

1. Check that `kubectl` and `helm` are installed and a cluster is reachable.
2. Install the **nginx ingress controller** if it is not already present.
3. Install **MicroFoundry** via the Helm chart from `ghcr.io` using the included `helm-values.yaml`.
4. Optionally install the **monitoring stack** (Prometheus, Grafana, Loki, Beyla).
5. Print access instructions.

### Script Options

| Flag | Description |
|------|-------------|
| `--with-monitoring` | Install the monitoring stack alongside MicroFoundry |
| `--version VERSION` | Override the chart version (default: the VERSION variable in the script) |
| `--namespace NS` | Override the target namespace (default: `microfoundry`) |
| `--dry-run` | Print commands without executing them |
| `-h`, `--help` | Show help |

---

## What Gets Deployed

### Core (always installed)

| Component | Namespace | Description |
|-----------|-----------|-------------|
| MicroFoundry | `microfoundry` | Admin server, platform controller |
| nginx ingress controller | `ingress-nginx` | Routes traffic to services via `*.cf-local.dev` |

### Monitoring (with `--with-monitoring`)

| Component | Namespace | Description |
|-----------|-----------|-------------|
| Prometheus | `monitoring` | Metrics collection and alerting |
| Grafana | `monitoring` | Dashboards and visualization |
| Alertmanager | `monitoring` | Alert routing and notification |
| Loki + Promtail | `monitoring` | Log aggregation |
| Beyla | `monitoring` | eBPF-based auto-instrumentation (RED metrics) |

---

## Accessing the Admin UI

### Option A -- Port-forward (quickest, no DNS required)

```bash
kubectl port-forward -n microfoundry svc/microfoundry 8080:8080
```

Then open http://localhost:8080 in your browser.

### Option B -- Ingress (requires DNS entry)

Add the following line to your hosts file:

- **macOS / Linux**: `/etc/hosts`
- **Windows**: `C:\Windows\System32\drivers\etc\hosts`

```
127.0.0.1  admin.cf-local.dev cf-local.dev grafana.cf-local.dev
```

Then open http://admin.cf-local.dev in your browser.

---

## Configuration

Copy the example config to your MicroFoundry config directory:

```bash
mkdir -p ~/.mf
cp mf.example.yaml ~/.mf/mf.yaml
```

Edit `~/.mf/mf.yaml` to customize your setup. The example is pre-configured for
Docker Desktop with the `docker-desktop` context. If you use kind or minikube,
update the `kubernetes.active` field and the corresponding cluster entry.

---

## Enabling Authentication (Keycloak)

MicroFoundry uses Keycloak as its OIDC identity provider. To enable authentication:

1. **Deploy Keycloak** using the MicroFoundry CLI:

   ```bash
   mf setup keycloak
   ```

2. **Edit your config** (`~/.mf/mf.yaml` or the in-cluster ConfigMap) and uncomment
   the `auth:` section:

   ```yaml
   auth:
     enabled: true
     issuer_url: "http://localhost:8180/realms/microfoundry"
     client_id: "mf-admin"
     client_secret: "<from-keycloak-admin-console>"
     redirect_url: "http://admin.cf-local.dev:8080/auth/callback"
     session_key: ""
     admin_base_url: "http://localhost:8180"
     admin_client_id: "admin-cli"
     admin_client_secret: "<from-keycloak-admin-console>"
     realm: "microfoundry"
   ```

3. **Restart MicroFoundry** (or the pod will pick up ConfigMap changes on next rollout):

   ```bash
   kubectl rollout restart deployment/microfoundry -n microfoundry
   ```

When using the Helm chart, you can also enable auth via values:

```bash
helm upgrade microfoundry oci://ghcr.io/younjinjeong/microfoundry/charts/microfoundry \
  -f helm-values.yaml \
  --set auth.enabled=true \
  --set auth.issuerUrl="http://localhost:8180/realms/microfoundry" \
  --set auth.clientSecret="<your-secret>" \
  -n microfoundry
```

---

## Enabling Monitoring

If you did not install monitoring during the initial setup, you can add it later:

```bash
# Re-run install with the monitoring flag
bash install.sh --with-monitoring
```

Or install it manually using the monitoring stack scripts:

```bash
cd ../../monitoring
bash install.sh
```

### Accessing monitoring services

| Service | URL (ingress) | Port-forward command |
|---------|---------------|---------------------|
| Grafana | http://grafana.cf-local.dev | `kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80` |
| Prometheus | -- | `kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-prometheus 9090:9090` |
| Alertmanager | -- | `kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-alertmanager 9093:9093` |

Default Grafana credentials: **admin** / **microfoundry**

---

## Uninstalling

```bash
# Remove MicroFoundry
helm uninstall microfoundry -n microfoundry
kubectl delete namespace microfoundry

# Remove monitoring (if installed)
helm uninstall kube-prometheus -n monitoring
helm uninstall loki -n monitoring
kubectl delete namespace monitoring

# Remove nginx ingress controller
helm uninstall ingress-nginx -n ingress-nginx
kubectl delete namespace ingress-nginx
```

---

## Troubleshooting

**Pods stuck in Pending state**
- Check resource limits: `kubectl describe pod -n microfoundry`
- Ensure Docker Desktop has enough CPU and memory allocated (recommended: 4 CPU, 8 GB RAM)

**Ingress not working**
- Verify the ingress controller is running: `kubectl get pods -n ingress-nginx`
- Check ingress resources: `kubectl get ingress -n microfoundry`
- Verify your hosts file has the correct entries

**Cannot pull images**
- If behind a proxy, configure Docker Desktop proxy settings
- Verify you can reach `ghcr.io`: `curl -I https://ghcr.io`
