# MicroFoundry

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**輕量級 Kubernetes PaaS 平台** — 保留 CloudFoundry 開發者體驗，底層運行在雲原生基礎設施上。

MicroFoundry 以 Kubernetes 取代笨重的 BOSH/Diego 運行時，搭配託管雲服務與現代可觀測性工具。實現 `cf push` 風格的部署、服務綁定與日誌串流 — 全部由 Kubernetes、Prometheus、Loki 和 Grafana Beyla 驅動。

---

## 為什麼選擇 MicroFoundry？

| 問題 | 解決方案 |
|------|----------|
| **CF 太重** — BOSH + Diego + 20 多台 VM | **單一 Go 二進位檔**，任何 K8s 叢集皆可運行 |
| **遷移到 K8s 後失去 CF 開發體驗** | **`mf push` 如同 `cf push`** — 相同工作流程，底層為 K8s |
| **可觀測性需要修改程式碼** | **零程式碼 eBPF 指標**，透過 Grafana Beyla |
| **缺乏平台能見度** | **內建管理後台** — 48 個模板，全部伺服器端渲染 |
| **IAM 外掛式整合** | **Keycloak + OPA + SCIM v2**，OIDC 聯邦至 AWS/GCP/Azure |
| **多叢集管理困難** | **單一控制平面** — Docker Desktop、EKS、GKE、AKS |

**主要功能：**

- **56 個後端服務** — 10 個本地 K8s + 21 個 AWS + 12 個 GCP + 13 個 Azure，每個服務 3 種方案
- **OIDC CSP 聯邦** — 透過 Keycloak 代理的臨時憑證（AWS STS、GCP WIF、Azure FIC）
- **5 層 RBAC** — platform-admin → workspace-admin → org-admin → member → viewer
- **MCP Server** — AI 工具（Claude、Cursor）可直接部署和管理應用程式
- **雲端部署套件** — EKS、ECS Fargate、GKE、AKS、本地 K8s 的 Terraform 藍圖
- **跨平台發佈** — GoReleaser 支援 Linux/macOS/Windows，多架構 Docker，Helm OCI

---

## Kind 本地部署指南

以下為在 Kind 叢集上完整安裝與驗證 MicroFoundry 的步驟。

### 前置需求

| 工具 | 最低版本 | 安裝方式 |
|------|----------|----------|
| Go | 1.25+ | `brew install go` |
| Docker | 最新版 | [Docker Desktop](https://www.docker.com/products/docker-desktop/) 或 [Rancher Desktop](https://rancherdesktop.io/) |
| Kind | 0.20+ | `brew install kind` |
| kubectl | 1.27+ | `brew install kubectl` |
| Helm | 3.12+ | `brew install helm` |

### Step 1：建立 Kind 叢集

```bash
# 如有舊叢集，先刪除
kind get clusters
kind delete cluster --name <cluster-name>

# 建立新叢集
kind create cluster --name microfoundry

# 驗證叢集
kubectl cluster-info --context kind-microfoundry
```

### Step 2：建置專案

```bash
# 建置 Go binary
make build

# 建置 Docker image
docker build -t ghcr.io/younjinjeong/microfoundry/mf:0.1.0 .

# 載入 image 到 Kind 叢集
kind load docker-image ghcr.io/younjinjeong/microfoundry/mf:0.1.0 --name microfoundry
```

### Step 3：安裝 Nginx Ingress Controller

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace \
  --set controller.service.type=ClusterIP \
  --set controller.watchIngressWithoutClass=true \
  --wait --timeout 5m
```

### Step 4：部署 MicroFoundry

```bash
# 建立 namespace 並標記為 Helm 管理
kubectl create namespace microfoundry
kubectl label namespace microfoundry app.kubernetes.io/managed-by=Helm
kubectl annotate namespace microfoundry \
  meta.helm.sh/release-name=microfoundry \
  meta.helm.sh/release-namespace=microfoundry

# 使用本地 Helm chart 部署
helm upgrade --install microfoundry deploy/helm/microfoundry \
  --namespace microfoundry \
  --values deploy/packages/local-k8s/helm-values.yaml \
  --set config.kubernetes.active=kind-microfoundry \
  --set 'config.kubernetes.clusters.kind-microfoundry.name=kind-microfoundry' \
  --set 'config.kubernetes.clusters.kind-microfoundry.context=kind-microfoundry' \
  --set 'config.kubernetes.clusters.kind-microfoundry.namespace=microfoundry' \
  --set 'config.kubernetes.clusters.kind-microfoundry.domain=cf-local.dev' \
  --set 'config.kubernetes.clusters.kind-microfoundry.provider=kind' \
  --set 'config.kubernetes.clusters.kind-microfoundry.enabled=true' \
  --set 'config.kubernetes.clusters.kind-microfoundry.ingress_class=nginx' \
  --set image.pullPolicy=Never \
  --wait --timeout 5m
```

### Step 5：驗證部署

```bash
# 檢查 Pod 狀態（應為 1/1 Running）
kubectl get pods -n microfoundry

# 檢查服務
kubectl get svc -n microfoundry

# 檢查 Ingress
kubectl get ingress -n microfoundry

# 查看日誌，確認 in-cluster config 正常
kubectl logs -n microfoundry deployment/microfoundry --tail=20
```

預期輸出應包含：
```
[k8s] using in-cluster config (service account)
```

### Step 6：存取管理後台

**方式 A — Port-forward（最快，無需 DNS 設定）**

```bash
kubectl port-forward -n microfoundry svc/microfoundry 8080:8080
```

開啟瀏覽器：http://localhost:8080

**方式 B — Ingress（需設定 DNS）**

在 hosts 檔案中加入以下內容：

- **macOS / Linux**：`/etc/hosts`
- **Windows**：`C:\Windows\System32\drivers\etc\hosts`

```
127.0.0.1  admin.cf-local.dev cf-local.dev
```

開啟瀏覽器：http://admin.cf-local.dev

---

## 管理後台

內建管理後台（`mf admin`）提供應用程式生命週期管理、服務目錄、多叢集管理、可觀測性、密鑰管理、IAM 和平台設定。

**管理頁面：** Dashboard、Applications、Services、Secrets、Users & Orgs（5 分頁 IAM）、Clusters、Service Catalog、Registry、Webhooks、SMTP、Endpoints、Cloud Providers、Metrics & Alerts、Platform、Documentation

---

## 架構

```
                          ┌─────────────────────────────────┐
                          │         Developer / AI          │
                          └──────────┬──────────┬───────────┘
                                     │          │
                              ┌──────▽──┐  ┌────▽──────┐
                              │  CLI    │  │  MCP      │
                              │ mf push │  │  Server   │
                              └──────┬──┘  └────┬──────┘
                                     │          │
                              ┌──────▽──────────▽──────┐
                              │  MicroFoundry Admin    │
                              │  API + Dashboard       │
                              └─────────────┬──────────┘
                                            │
          ┌─────────────────────────────────┼─────────────────────────────────┐
          │                                 │                                 │
  ┌───────▽────────┐              ┌─────────▽──────────┐            ┌────────▽────────┐
  │  Build System  │              │  Kubernetes API     │            │  Observability  │
  │  Dockerfile    │              │  Deployments        │            │  Prometheus     │
  │  CNB/Paketo    │              │  Services/Ingress   │            │  Grafana + Loki │
  └────────────────┘              │  Secrets/ConfigMaps │            │  Beyla (eBPF)   │
                                  └─────────┬──────────┘            └─────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
            ┌───────▽───────┐       ┌───────▽───────┐      ┌───────▽───────┐
            │ API Gateway   │       │ Backing       │      │ IAM           │
            │ Kong/Nginx/   │       │ Services      │      │ Keycloak OIDC │
            │ AWS API GW    │       │ 56 types      │      │ OPA + SCIM v2 │
            └───────────────┘       └───────────────┘      └───────────────┘
```

詳細架構設計請參閱 [Architecture](docs/architecture.md)。

---

## 服務目錄

56 個後端服務，橫跨 4 個供應商，每個服務提供 3 種方案（small / medium / large）：

| 供應商 | 數量 | 服務 |
|--------|------|------|
| **本地 K8s** | 10 | MariaDB、PostgreSQL、ClickHouse、Redis、Memcached、RabbitMQ、ActiveMQ、MinIO、Kong、Nginx |
| **AWS** | 21 | RDS（PostgreSQL、MySQL、MariaDB）、Aurora、DynamoDB、DocumentDB、Redshift、ElastiCache、SQS、SNS、MQ、MSK、Kinesis、S3、OpenSearch、Bedrock、SageMaker、MediaConvert、IVS |
| **GCP** | 12 | Cloud SQL（PostgreSQL、MySQL）、AlloyDB、Spanner、Firestore、Bigtable、BigQuery、Memorystore、Pub/Sub、Cloud Storage、Vertex AI、Transcoder |
| **Azure** | 13 | Database（PostgreSQL、MySQL）、SQL Database、Cosmos DB、Synapse、Cache for Redis、Service Bus、Event Hubs、Blob Storage、AI Search、Azure OpenAI、ML、Media Services |

```bash
mf catalog                                # 列出所有服務
mf create-service postgresql small my-db  # 建立服務
mf bind-service hello-world my-db         # 綁定服務 → VCAP_SERVICES
```

雲端供應商支援透過 Keycloak 的 **OIDC 聯邦** — 無需靜態憑證。詳見 [Admin Guide](docs/admin-guide.md)。

---

## CLI 指令

| 指令 | 說明 |
|------|------|
| `mf push [app]` | 從原始碼建置並部署（Dockerfile 或 CNB） |
| `mf apps` / `mf app [name]` | 列出應用程式 / 顯示應用程式詳情 |
| `mf logs [app]` | 串流或取得應用程式日誌 |
| `mf scale [app] -i N` | 擴縮應用程式實例 |
| `mf delete [app]` | 刪除應用程式並清理路由 |
| `mf catalog` | 依供應商列出可用服務 |
| `mf create-service` / `mf delete-service` | 建立 / 刪除後端服務 |
| `mf services` / `mf bind-service` / `mf unbind-service` | 列出 / 綁定 / 解除綁定服務 |
| `mf secrets` / `mf create-secret` / `mf delete-secret` | 管理密鑰 |
| `mf admin` | 啟動管理後台 |
| `mf setup keycloak` / `keycloak-realm` / `keycloak-idp` | 認證設定 |
| `mf users` / `mf create-user` | Keycloak 使用者管理 |
| `mf orgs` / `mf create-org` | 組織管理 |
| `mf auth login` | OIDC 認證 |

### MCP Server

9 個 AI 整合工具：`mf_push`、`mf_apps`、`mf_logs`、`mf_scale`、`mf_delete`、`mf_create_service`、`mf_bind_service`、`mf_routes`、`mf_env`

---

## 雲端部署

提供所有主要雲端供應商的 Terraform 部署套件。預設成本最佳化 — 即使 MicroFoundry 停機，現有應用程式仍在 K8s 上持續運行。

| 平台 | 架構 | 預估費用 |
|------|------|----------|
| **AWS ECS Fargate + EKS** | Fargate 控制平面 + EKS 工作負載 | 約 $108/月 |
| **AWS ECS Fargate only** | 連接現有 EKS 叢集 | 約 $20/月 |
| **AWS EKS** | Helm 安裝至 EKS | 依叢集而定 |
| **GCP GKE** | Helm 安裝至 GKE Autopilot/Standard | 依叢集而定 |
| **Azure AKS** | Helm 安裝至 AKS | 依叢集而定 |
| **本地 K8s** | Docker Desktop / Kind / minikube | 免費 |

詳見 [deploy/packages/](deploy/packages/) 各供應商的部署指南。

---

## 監控堆疊（選用）

```bash
# 透過安裝腳本安裝
bash deploy/packages/local-k8s/install.sh --with-monitoring

# 或手動安裝
bash deploy/monitoring/install.sh
```

### 監控服務存取

| 服務 | Port-forward 指令 |
|------|-------------------|
| Grafana | `kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80` |
| Prometheus | `kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-prometheus 9090:9090` |
| Alertmanager | `kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-alertmanager 9093:9093` |

預設 Grafana 帳號密碼：**admin** / **microfoundry**

---

## 認證設定（Keycloak）

MicroFoundry 使用 Keycloak 作為 OIDC 身份提供者。

```bash
# 部署 Keycloak
mf setup keycloak

# 設定 Realm
mf setup keycloak-realm --url http://localhost:8180
```

編輯設定檔（`~/.mf/mf.yaml`）並啟用 auth 區段：

```yaml
auth:
  enabled: true
  issuer_url: "http://localhost:8180/realms/microfoundry"
  client_id: "mf-admin"
  client_secret: "<從 Keycloak 管理介面取得>"
  redirect_url: "http://admin.cf-local.dev:8080/auth/callback"
  realm: "microfoundry"
```

重新啟動 MicroFoundry 使設定生效：

```bash
kubectl rollout restart deployment/microfoundry -n microfoundry
```

---

## 技術堆疊

| 層級 | 技術 | 用途 |
|------|------|------|
| **語言** | Go 1.25 | API 伺服器、CLI、MCP Server |
| **CLI** | Cobra + Viper | 指令與設定管理 |
| **執行環境** | Kubernetes | 排程與編排 |
| **建置** | Cloud Native Buildpacks | 原始碼到容器映像檔 |
| **Ingress** | Kong / Nginx / Traefik / AWS API GW | 可插拔閘道器，支援 WebSocket/gRPC |
| **TLS** | mkcert | 本地 HTTPS，`.dev` 網域 |
| **指標** | Prometheus + Grafana + Beyla (eBPF) | 零程式碼自動化指標蒐集 |
| **日誌** | Promtail + Loki | 日誌聚合 |
| **認證** | Keycloak + go-oidc + OPA | OIDC + Rego 政策 + SCIM v2 |
| **UI** | Go templates + HTMX + Tailwind CSS | 伺服器端渲染，無 JS 建置步驟 |
| **IaC** | Terraform | 雲端資源供應 |
| **AI** | Model Context Protocol (MCP) | AI 工具整合 |
| **CSP** | AWS STS + GCP WIF + Azure FIC | OIDC 憑證聯邦 |

---

## 解除安裝

```bash
# 移除 MicroFoundry
helm uninstall microfoundry -n microfoundry
kubectl delete namespace microfoundry

# 移除監控（如已安裝）
helm uninstall kube-prometheus -n monitoring
helm uninstall loki -n monitoring
kubectl delete namespace monitoring

# 移除 nginx ingress controller
helm uninstall ingress-nginx -n ingress-nginx
kubectl delete namespace ingress-nginx

# 刪除 Kind 叢集
kind delete cluster --name microfoundry
```

---

## 疑難排解

**Pod 卡在 Pending 狀態**
- 檢查資源限制：`kubectl describe pod -n microfoundry`
- 確保 Docker 分配足夠的 CPU 和記憶體（建議：4 CPU、8 GB RAM）

**Ingress 無法運作**
- 確認 ingress controller 正在運行：`kubectl get pods -n ingress-nginx`
- 檢查 ingress 資源：`kubectl get ingress -n microfoundry`
- 確認 hosts 檔案已正確設定

**Pod 因 kubeconfig 錯誤重啟**
- 確認 Pod 日誌中出現 `using in-cluster config (service account)`
- 若出現 `loading kubeconfig: stat /root/.kube/config: no such file or directory`，表示 in-cluster config 未正確啟用

**無法拉取映像檔**
- Kind 部署時使用 `kind load docker-image` 載入本地映像檔
- Helm values 設定 `image.pullPolicy=Never`

---

## 文件

| 文件 | 說明 |
|------|------|
| [User Manual](docs/user-manual.md) | 部署與管理應用程式 |
| [Architecture](docs/architecture.md) | 技術設計與專案結構 |
| [Admin Guide](docs/admin-guide.md) | 管理後台頁面與 API 參考（100+ 端點） |
| [Development Workflow](docs/development-workflow.md) | Human-AI 協作開發流程 |
| [Observability & Capacity](docs/observability-capacity.md) | 監控與容量規劃 |

---

## 貢獻

```bash
make hooks              # 安裝 pre-commit hooks
go build ./...          # 必須通過
go vet ./...            # 必須通過
go test ./...           # 執行測試
```

---

## 授權

See [LICENSE](LICENSE).
