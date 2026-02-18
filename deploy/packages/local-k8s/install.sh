#!/bin/bash
set -euo pipefail

# ---------------------------------------------------------------------------
# MicroFoundry — Local Kubernetes Installer
# Target: Docker Desktop / kind / minikube
# ---------------------------------------------------------------------------

VERSION="0.1.0"  # Replaced during packaging by CI/CD

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="microfoundry"
RELEASE_NAME="microfoundry"
CHART_REF="oci://ghcr.io/younjinjeong/microfoundry/charts/microfoundry"
HELM_VALUES="${SCRIPT_DIR}/helm-values.yaml"
MONITORING_DIR="$(cd "${SCRIPT_DIR}/../../monitoring" 2>/dev/null && pwd || echo "")"

WITH_MONITORING=false

# ---------------------------------------------------------------------------
# Color helpers (degrade gracefully when not a terminal)
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  BOLD="\033[1m"
  GREEN="\033[0;32m"
  YELLOW="\033[0;33m"
  RED="\033[0;31m"
  RESET="\033[0m"
else
  BOLD="" GREEN="" YELLOW="" RED="" RESET=""
fi

info()  { echo -e "${GREEN}[INFO]${RESET}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
error() { echo -e "${RED}[ERROR]${RESET} $*" >&2; }

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------
usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Install MicroFoundry on a local Kubernetes cluster.

Options:
  --with-monitoring    Also install the monitoring stack
                       (Prometheus, Grafana, Loki, Beyla)
  --version VERSION    Override the MicroFoundry chart version
                       (default: ${VERSION})
  --namespace NS       Override the target namespace
                       (default: ${NAMESPACE})
  --dry-run            Print what would be executed without running it
  -h, --help           Show this help message

Examples:
  $(basename "$0")
  $(basename "$0") --with-monitoring
  $(basename "$0") --version 0.2.0 --with-monitoring
EOF
  exit 0
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-monitoring)
      WITH_MONITORING=true
      shift
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    -h|--help)
      usage
      ;;
    *)
      error "Unknown option: $1"
      usage
      ;;
  esac
done

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------
check_command() {
  local cmd="$1"
  local install_hint="$2"
  if ! command -v "$cmd" &>/dev/null; then
    error "'${cmd}' is not installed or not in PATH."
    echo "  Install it: ${install_hint}"
    exit 1
  fi
}

echo ""
echo -e "${BOLD}=== MicroFoundry Local Kubernetes Installer (v${VERSION}) ===${RESET}"
echo ""

info "Checking prerequisites..."

check_command "kubectl" "https://kubernetes.io/docs/tasks/tools/"
check_command "helm"    "https://helm.sh/docs/intro/install/"

# Verify cluster connectivity
if ! kubectl cluster-info &>/dev/null; then
  error "Cannot connect to a Kubernetes cluster."
  echo "  Make sure Docker Desktop / kind / minikube is running and kubectl is configured."
  exit 1
fi

CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "unknown")
info "Connected to Kubernetes cluster (context: ${CURRENT_CONTEXT})"

# ---------------------------------------------------------------------------
# Ask about monitoring if --with-monitoring was not passed
# ---------------------------------------------------------------------------
if [ "$WITH_MONITORING" = false ] && [ -t 0 ]; then
  echo ""
  read -r -p "Install monitoring stack (Prometheus/Grafana/Loki/Beyla)? [y/N] " answer
  case "$answer" in
    [yY]|[yY][eE][sS]) WITH_MONITORING=true ;;
    *) WITH_MONITORING=false ;;
  esac
fi

# ---------------------------------------------------------------------------
# Dry-run guard
# ---------------------------------------------------------------------------
run() {
  if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN]${RESET} $*"
  else
    "$@"
  fi
}

# ---------------------------------------------------------------------------
# Step 1 — Install nginx ingress controller (if not present)
# ---------------------------------------------------------------------------
echo ""
info "Step 1/4: Checking nginx ingress controller..."

if kubectl get ns ingress-nginx &>/dev/null; then
  info "  nginx ingress controller namespace already exists — skipping."
else
  info "  Installing nginx ingress controller..."
  run helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
  run helm repo update
  run helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
    --namespace ingress-nginx --create-namespace \
    --set controller.service.type=ClusterIP \
    --set controller.watchIngressWithoutClass=true \
    --wait --timeout 5m
fi

# ---------------------------------------------------------------------------
# Step 2 — Install MicroFoundry
# ---------------------------------------------------------------------------
info "Step 2/4: Installing MicroFoundry (${RELEASE_NAME})..."

if [ ! -f "$HELM_VALUES" ]; then
  error "Helm values file not found: ${HELM_VALUES}"
  exit 1
fi

run helm upgrade --install "$RELEASE_NAME" "$CHART_REF" \
  --version "$VERSION" \
  --namespace "$NAMESPACE" --create-namespace \
  --values "$HELM_VALUES" \
  --wait --timeout 5m

# ---------------------------------------------------------------------------
# Step 3 — Monitoring stack (optional)
# ---------------------------------------------------------------------------
if [ "$WITH_MONITORING" = true ]; then
  info "Step 3/4: Installing monitoring stack..."

  if [ -n "$MONITORING_DIR" ] && [ -f "${MONITORING_DIR}/install.sh" ]; then
    run bash "${MONITORING_DIR}/install.sh"
  else
    warn "Monitoring install script not found at deploy/monitoring/install.sh"
    warn "Installing monitoring components inline..."

    # Add Helm repos
    run helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    run helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
    run helm repo update

    # Create monitoring namespace
    run kubectl create namespace monitoring 2>/dev/null || true

    # Install kube-prometheus-stack
    info "  Installing kube-prometheus-stack..."
    run helm upgrade --install kube-prometheus prometheus-community/kube-prometheus-stack \
      --namespace monitoring \
      --set grafana.adminPassword=microfoundry \
      --set grafana.ingress.enabled=true \
      --set grafana.ingress.ingressClassName=nginx \
      --set 'grafana.ingress.hosts[0]=grafana.cf-local.dev' \
      --set prometheus.prometheusSpec.retention=7d \
      --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
      --set kubeEtcd.enabled=false \
      --set kubeControllerManager.enabled=false \
      --set kubeScheduler.enabled=false \
      --set kubeProxy.enabled=false \
      --wait --timeout 10m

    # Install Loki
    info "  Installing Loki + Promtail..."
    run helm upgrade --install loki grafana/loki-stack \
      --namespace monitoring \
      --set loki.config.auth_enabled=false \
      --set grafana.enabled=false \
      --set prometheus.enabled=false \
      --wait --timeout 5m
  fi
else
  info "Step 3/4: Monitoring stack — skipped (use --with-monitoring to install)"
fi

# ---------------------------------------------------------------------------
# Step 4 — Verify deployment
# ---------------------------------------------------------------------------
info "Step 4/4: Verifying deployment..."

if [ "$DRY_RUN" = false ]; then
  echo ""
  kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null || true
  echo ""
fi

# ---------------------------------------------------------------------------
# Access instructions
# ---------------------------------------------------------------------------
echo ""
echo -e "${BOLD}=== MicroFoundry installed successfully ===${RESET}"
echo ""
echo "  Namespace:  ${NAMESPACE}"
echo "  Context:    ${CURRENT_CONTEXT}"
echo "  Version:    ${VERSION}"
echo ""
echo -e "${BOLD}-- Access the Admin UI --${RESET}"
echo ""
echo "  Option A: Port-forward (quickest)"
echo "    kubectl port-forward -n ${NAMESPACE} svc/${RELEASE_NAME} 8080:8080"
echo "    Then open: http://localhost:8080"
echo ""
echo "  Option B: Ingress (requires DNS / /etc/hosts entry)"
echo "    Add to /etc/hosts (or C:\\Windows\\System32\\drivers\\etc\\hosts on Windows):"
echo "      127.0.0.1  admin.cf-local.dev cf-local.dev"
echo "    Then open: http://admin.cf-local.dev"
echo ""

if [ "$WITH_MONITORING" = true ]; then
  echo -e "${BOLD}-- Monitoring --${RESET}"
  echo ""
  echo "  Grafana:      http://grafana.cf-local.dev  (admin / microfoundry)"
  echo "                or: kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80"
  echo "  Prometheus:   kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-prometheus 9090:9090"
  echo "  Alertmanager: kubectl port-forward -n monitoring svc/kube-prometheus-kube-prome-alertmanager 9093:9093"
  echo ""
  echo "  Add to /etc/hosts: 127.0.0.1  grafana.cf-local.dev"
  echo ""
fi

echo -e "${BOLD}-- Next steps --${RESET}"
echo ""
echo "  1. Copy mf.example.yaml to ~/.mf/mf.yaml and adjust as needed"
echo "  2. Run 'mf setup tls' to generate TLS certificates"
echo "  3. Run 'mf setup keycloak' to deploy Keycloak for authentication"
echo ""
