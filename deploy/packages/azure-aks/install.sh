#!/bin/bash
# -----------------------------------------------------------------------------
# MicroFoundry - Azure AKS Installation Script
#
# Provisions an AKS cluster with all supporting infrastructure via Terraform,
# configures kubectl, and prints access information.
#
# Prerequisites: terraform, az (Azure CLI), kubectl, helm
# -----------------------------------------------------------------------------
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${SCRIPT_DIR}/terraform"

# -- Colors (disabled when stdout is not a terminal) --------------------------
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[0;33m'
  CYAN='\033[0;36m'
  NC='\033[0m'
else
  RED='' GREEN='' YELLOW='' CYAN='' NC=''
fi

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }

# -- Prerequisite checks ------------------------------------------------------
check_command() {
  if ! command -v "$1" &>/dev/null; then
    error "Required command not found: $1"
    echo "  Install instructions: $2"
    return 1
  fi
  ok "$1 found ($(command -v "$1"))"
}

info "Checking prerequisites..."
MISSING=0
check_command terraform "https://developer.hashicorp.com/terraform/install"          || MISSING=1
check_command az        "https://learn.microsoft.com/cli/azure/install-azure-cli"    || MISSING=1
check_command kubectl   "https://kubernetes.io/docs/tasks/tools/"                    || MISSING=1
check_command helm      "https://helm.sh/docs/intro/install/"                        || MISSING=1

if [ "$MISSING" -ne 0 ]; then
  error "One or more prerequisites are missing. Install them and re-run this script."
  exit 1
fi

# -- Verify Azure CLI login ---------------------------------------------------
info "Verifying Azure CLI authentication..."
if ! az account show &>/dev/null; then
  warn "Not logged in to Azure CLI. Running 'az login'..."
  az login
fi
AZURE_SUBSCRIPTION="$(az account show --query 'name' -o tsv)"
AZURE_SUBSCRIPTION_ID="$(az account show --query 'id' -o tsv)"
ok "Azure subscription: ${AZURE_SUBSCRIPTION} (${AZURE_SUBSCRIPTION_ID})"

# -- Terraform init & apply ---------------------------------------------------
info "Initializing Terraform..."
terraform -chdir="${TF_DIR}" init

info "Planning infrastructure changes..."
terraform -chdir="${TF_DIR}" plan -out=tfplan

echo ""
echo -e "${YELLOW}Review the plan above. Proceed with apply?${NC}"
read -rp "Type 'yes' to continue: " CONFIRM
if [ "${CONFIRM}" != "yes" ]; then
  info "Aborted by user."
  exit 0
fi

info "Applying Terraform configuration..."
terraform -chdir="${TF_DIR}" apply tfplan

# Clean up plan file
rm -f "${TF_DIR}/tfplan"

# -- Configure kubectl ---------------------------------------------------------
info "Configuring kubectl..."
KUBECONFIG_CMD="$(terraform -chdir="${TF_DIR}" output -raw kubeconfig_command)"
eval "${KUBECONFIG_CMD}"
ok "kubectl configured for AKS cluster."

# Verify connectivity
info "Verifying cluster connectivity..."
kubectl cluster-info
kubectl get nodes

# -- Print access information --------------------------------------------------
echo ""
echo "==========================================================================="
echo "  MicroFoundry -- Azure AKS Deployment Complete"
echo "==========================================================================="
echo ""
echo "  AKS Cluster:     $(terraform -chdir="${TF_DIR}" output -raw aks_cluster_name)"
echo "  Resource Group:   $(terraform -chdir="${TF_DIR}" output -raw aks_resource_group)"
echo "  ACR Login Server: $(terraform -chdir="${TF_DIR}" output -raw acr_login_server)"
echo "  App Gateway IP:   $(terraform -chdir="${TF_DIR}" output -raw appgw_public_ip)"
echo "  Admin URL:        $(terraform -chdir="${TF_DIR}" output -raw mf_admin_url)"
echo ""
echo "  kubectl config:   ${KUBECONFIG_CMD}"
echo ""
echo "  Next steps:"
echo "    1. Point your DNS records to the Application Gateway IP above."
echo "    2. Configure TLS certificates (Azure Key Vault or manual secret)."
echo "    3. Set up authentication in the MicroFoundry admin UI."
echo "    4. (Optional) Push images to ACR:"
echo "       az acr login --name $(terraform -chdir="${TF_DIR}" output -raw acr_login_server | cut -d. -f1)"
echo ""
echo "==========================================================================="
