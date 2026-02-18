# MicroFoundry -- Azure AKS Deployment Package

Deploy MicroFoundry on Azure Kubernetes Service (AKS) with Terraform.

This package provisions a production-ready AKS cluster with Application Gateway
Ingress Controller (AGIC), Azure Container Registry (ACR), Workload Identity,
and optional Azure Monitor integration.

## Architecture

```
Internet
   |
   v
Azure Application Gateway (AGIC)   <-- Public IP, TLS termination
   |
   v
AKS Cluster (azure CNI, calico)
   +-- microfoundry namespace
   |      +-- MicroFoundry (Helm release)
   |      +-- ServiceAccount (Workload Identity -> Managed Identity)
   |
   +-- monitoring (Azure Monitor / Log Analytics)
   |
   v
Azure Container Registry (ACR)     <-- AcrPull role assigned to AKS kubelet
```

## Prerequisites

| Tool      | Minimum Version | Install Link                                                     |
|-----------|-----------------|------------------------------------------------------------------|
| Terraform | >= 1.5          | https://developer.hashicorp.com/terraform/install                |
| Azure CLI | >= 2.50         | https://learn.microsoft.com/cli/azure/install-azure-cli          |
| kubectl   | >= 1.28         | https://kubernetes.io/docs/tasks/tools/                          |
| Helm      | >= 3.12         | https://helm.sh/docs/intro/install/                              |

You must be logged in to Azure CLI (`az login`) with a subscription that has
permission to create resource groups, AKS clusters, ACR, VNets, and Application
Gateways.

## Quickstart

### 1. Clone and navigate

```bash
cd microfoundry/deploy/packages/azure-aks
```

### 2. Create a Terraform variables file

```bash
cat > terraform/terraform.tfvars <<'EOF'
domain     = "apps.example.com"
mf_version = "0.1.0"
location   = "eastus"

# Optional overrides
# project_name          = "microfoundry"
# aks_vm_size           = "Standard_D2s_v3"
# aks_node_count        = 2
# aks_kubernetes_version = "1.29"
# enable_monitoring     = true
# tags = {
#   Team = "platform"
# }
EOF
```

### 3. Run the installer

```bash
chmod +x install.sh
./install.sh
```

The script will:
1. Check that all prerequisites are installed.
2. Verify Azure CLI authentication (prompts `az login` if needed).
3. Run `terraform init` and `terraform plan` for review.
4. Apply the Terraform configuration after confirmation.
5. Configure `kubectl` for the new AKS cluster.
6. Print cluster details and next steps.

### 4. Manual Terraform workflow (alternative)

```bash
cd terraform
terraform init
terraform plan -var 'domain=apps.example.com' -var 'mf_version=0.1.0'
terraform apply -var 'domain=apps.example.com' -var 'mf_version=0.1.0'

# Configure kubectl
eval "$(terraform output -raw kubeconfig_command)"
```

## Post-deployment steps

### DNS

Point your domain to the Application Gateway public IP:

```bash
# Get the public IP
terraform -chdir=terraform output -raw appgw_public_ip

# Create DNS records:
#   A  admin.apps.example.com  ->  <appgw_public_ip>
#   A  *.apps.example.com      ->  <appgw_public_ip>
```

### TLS certificates

Option A -- Azure Key Vault integration (recommended):

```bash
# Store certificate in Key Vault and reference it via AGIC annotations.
# See: https://learn.microsoft.com/azure/application-gateway/key-vault-certs
```

Option B -- Manual Kubernetes secret:

```bash
kubectl create secret tls mf-tls-wildcard \
  --cert=cert.pem --key=key.pem -n microfoundry
```

Then update the ingress TLS section in `helm-values.yaml`.

### Authentication

Configure Keycloak or another OIDC provider, then update the `auth` section
in MicroFoundry settings via the admin UI or by editing `mf.example.yaml`.

### Push images to ACR

```bash
ACR_NAME=$(terraform -chdir=terraform output -raw acr_login_server | cut -d. -f1)
az acr login --name "$ACR_NAME"
docker tag myapp:latest "${ACR_NAME}.azurecr.io/myapp:latest"
docker push "${ACR_NAME}.azurecr.io/myapp:latest"
```

## File structure

```
azure-aks/
  terraform/
    versions.tf       Terraform and provider version constraints
    variables.tf      Input variables with defaults and validation
    main.tf           AKS cluster, VNet, ACR, AGIC, monitoring, Helm release
    outputs.tf        Cluster access info and resource identifiers
  helm-values.yaml    Helm values for MicroFoundry on AKS
  mf.example.yaml     Example MicroFoundry configuration file
  install.sh          One-command installation script
  README.md           This file
```

## Tear down

```bash
cd terraform
terraform destroy
```

This removes all Azure resources created by this package, including the AKS
cluster, ACR, Application Gateway, VNet, and Log Analytics workspace.

## Troubleshooting

**Terraform state lock**: If a previous run was interrupted, unlock with:
```bash
terraform -chdir=terraform force-unlock <LOCK_ID>
```

**AGIC not routing traffic**: Verify the ingress resource exists and the
Application Gateway backend health is healthy:
```bash
kubectl get ingress -n microfoundry
az network application-gateway show-backend-health \
  --resource-group rg-microfoundry --name agw-microfoundry
```

**ACR pull failures**: Confirm the AcrPull role assignment:
```bash
az role assignment list --scope $(terraform -chdir=terraform output -raw acr_login_server) \
  --query "[?roleDefinitionName=='AcrPull']"
```

**Azure Monitor not collecting data**: Check the OMS agent pods:
```bash
kubectl get pods -n kube-system -l component=oms-agent
```
