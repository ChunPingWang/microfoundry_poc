#!/bin/bash
# -----------------------------------------------------------------------------
# MicroFoundry - AWS EKS Installation Script
#
# This script provisions the full MicroFoundry stack on AWS EKS:
#   - VPC with public/private subnets across 3 AZs
#   - EKS cluster with managed node group
#   - ECR repository for application images
#   - AWS Load Balancer Controller
#   - CloudWatch Container Insights (optional)
#   - MicroFoundry via Helm
#
# Usage:
#   ./install.sh
#
# Prerequisites:
#   - terraform >= 1.5
#   - aws CLI v2 (authenticated)
#   - kubectl
#   - helm >= 3.0
# -----------------------------------------------------------------------------
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${SCRIPT_DIR}/terraform"

# -- Colors for output -------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
fatal() { error "$*"; exit 1; }

# -- Prerequisite checks -----------------------------------------------------
check_command() {
  local cmd="$1"
  local install_hint="${2:-}"
  if ! command -v "$cmd" &>/dev/null; then
    fatal "'${cmd}' is required but not found. ${install_hint}"
  fi
  ok "${cmd} found: $(command -v "$cmd")"
}

info "Checking prerequisites..."
echo ""
check_command "terraform"  "Install from https://developer.hashicorp.com/terraform/downloads"
check_command "aws"        "Install from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
check_command "kubectl"    "Install from https://kubernetes.io/docs/tasks/tools/"
check_command "helm"       "Install from https://helm.sh/docs/intro/install/"
echo ""

# Verify AWS credentials are configured
info "Verifying AWS credentials..."
if ! aws sts get-caller-identity &>/dev/null; then
  fatal "AWS credentials are not configured. Run 'aws configure' or export AWS_PROFILE."
fi
AWS_ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
AWS_REGION="$(aws configure get region 2>/dev/null || echo "us-east-1")"
ok "Authenticated as account ${AWS_ACCOUNT_ID} in region ${AWS_REGION}"
echo ""

# -- Terraform ----------------------------------------------------------------
info "Initializing Terraform..."
terraform -chdir="${TERRAFORM_DIR}" init

echo ""
info "Planning infrastructure changes..."
terraform -chdir="${TERRAFORM_DIR}" plan -out=tfplan

echo ""
echo "----------------------------------------------------------------------"
echo "  Review the plan above. Terraform will create the following:"
echo "    - VPC with public and private subnets (3 AZs)"
echo "    - EKS cluster with managed node group"
echo "    - ECR repository"
echo "    - AWS Load Balancer Controller"
echo "    - CloudWatch Container Insights (if enabled)"
echo "    - MicroFoundry Helm release"
echo "----------------------------------------------------------------------"
echo ""

read -rp "Proceed with deployment? (y/N): " CONFIRM
if [[ "${CONFIRM}" != "y" && "${CONFIRM}" != "Y" ]]; then
  info "Deployment cancelled."
  exit 0
fi

echo ""
info "Applying Terraform configuration (this may take 15-25 minutes)..."
terraform -chdir="${TERRAFORM_DIR}" apply tfplan

# -- Configure kubectl --------------------------------------------------------
echo ""
info "Configuring kubectl..."
KUBECONFIG_CMD="$(terraform -chdir="${TERRAFORM_DIR}" output -raw kubeconfig_command)"
eval "${KUBECONFIG_CMD}"
ok "kubectl configured for EKS cluster"

# Verify connectivity
info "Verifying cluster connectivity..."
if kubectl get nodes &>/dev/null; then
  ok "Successfully connected to the EKS cluster"
  kubectl get nodes
else
  warn "Could not connect to the cluster. Run the following command manually:"
  echo "  ${KUBECONFIG_CMD}"
fi

# -- Print access information -------------------------------------------------
echo ""
echo "======================================================================"
echo "  MicroFoundry on AWS EKS - Deployment Complete"
echo "======================================================================"
echo ""

CLUSTER_NAME="$(terraform -chdir="${TERRAFORM_DIR}" output -raw eks_cluster_name 2>/dev/null || echo "N/A")"
CLUSTER_ENDPOINT="$(terraform -chdir="${TERRAFORM_DIR}" output -raw eks_cluster_endpoint 2>/dev/null || echo "N/A")"
ECR_URL="$(terraform -chdir="${TERRAFORM_DIR}" output -raw ecr_repository_url 2>/dev/null || echo "N/A")"
ADMIN_URL="$(terraform -chdir="${TERRAFORM_DIR}" output -raw mf_admin_url 2>/dev/null || echo "N/A")"

echo "  EKS Cluster:     ${CLUSTER_NAME}"
echo "  Cluster Endpoint: ${CLUSTER_ENDPOINT}"
echo "  ECR Repository:  ${ECR_URL}"
echo "  Admin UI:        ${ADMIN_URL}"
echo ""
echo "  Configure kubectl:"
echo "    ${KUBECONFIG_CMD}"
echo ""
echo "  View MicroFoundry pods:"
echo "    kubectl get pods -n microfoundry"
echo ""
echo "  View MicroFoundry logs:"
echo "    kubectl logs -n microfoundry -l app.kubernetes.io/name=microfoundry -f"
echo ""
echo "  To destroy all resources:"
echo "    terraform -chdir=${TERRAFORM_DIR} destroy"
echo ""
echo "======================================================================"
