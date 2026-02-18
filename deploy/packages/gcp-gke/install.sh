#!/bin/bash
set -euo pipefail

##############################################################################
# MicroFoundry — GCP GKE Installation Script
#
# Provisions a GKE cluster with all supporting infrastructure and deploys
# MicroFoundry via Helm. Requires: terraform, gcloud, kubectl, helm.
##############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${SCRIPT_DIR}/terraform"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log()  { echo "==> $*"; }
err()  { echo "ERROR: $*" >&2; exit 1; }

check_command() {
  if ! command -v "$1" &>/dev/null; then
    err "'$1' is not installed. Please install it before running this script."
  fi
}

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------

log "Checking prerequisites..."

check_command terraform
check_command gcloud
check_command kubectl
check_command helm

# Verify gcloud is authenticated
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -1 | grep -q '.'; then
  err "No active gcloud account found. Run 'gcloud auth login' first."
fi

ACTIVE_PROJECT=$(gcloud config get-value project 2>/dev/null || true)
if [ -z "${ACTIVE_PROJECT}" ]; then
  log "WARNING: No default GCP project set. Terraform will use the project_id variable."
fi

log "All prerequisites satisfied."
echo ""

# ---------------------------------------------------------------------------
# Terraform Init & Apply
# ---------------------------------------------------------------------------

log "Initializing Terraform..."
terraform -chdir="${TF_DIR}" init

echo ""
log "Planning infrastructure changes..."
terraform -chdir="${TF_DIR}" plan -out=tfplan

echo ""
read -rp "Apply the plan above? [y/N] " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
  log "Aborted by user."
  exit 0
fi

echo ""
log "Applying Terraform plan..."
terraform -chdir="${TF_DIR}" apply tfplan

echo ""

# ---------------------------------------------------------------------------
# Configure kubectl
# ---------------------------------------------------------------------------

log "Configuring kubectl credentials..."
KUBECONFIG_CMD=$(terraform -chdir="${TF_DIR}" output -raw kubeconfig_command)
eval "${KUBECONFIG_CMD}"

log "Verifying cluster connectivity..."
kubectl cluster-info

echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

CLUSTER_NAME=$(terraform -chdir="${TF_DIR}" output -raw gke_cluster_name)
AR_URL=$(terraform -chdir="${TF_DIR}" output -raw artifact_registry_url)
ADMIN_URL=$(terraform -chdir="${TF_DIR}" output -raw mf_admin_url)
INGRESS_IP=$(terraform -chdir="${TF_DIR}" output -raw ingress_ip)

echo ""
echo "============================================================"
echo "  MicroFoundry on GKE — Deployment Complete"
echo "============================================================"
echo ""
echo "  GKE Cluster:        ${CLUSTER_NAME}"
echo "  Artifact Registry:  ${AR_URL}"
echo "  Ingress IP:         ${INGRESS_IP}"
echo "  Admin UI:           ${ADMIN_URL}"
echo ""
echo "  kubectl context has been configured automatically."
echo ""
echo "  Next steps:"
echo "    1. Point your DNS A record to ${INGRESS_IP}"
echo "    2. Open ${ADMIN_URL} in your browser"
echo "    3. (Optional) Configure TLS with a Google-managed certificate"
echo "    4. (Optional) Configure authentication in mf.yaml"
echo ""
echo "  To tear down all resources:"
echo "    terraform -chdir=${TF_DIR} destroy"
echo ""
echo "============================================================"
