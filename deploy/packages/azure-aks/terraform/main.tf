# -----------------------------------------------------------------------------
# MicroFoundry - Azure AKS Deployment Package
# Main Infrastructure Configuration
# -----------------------------------------------------------------------------

locals {
  # Azure naming convention: <project>-<resource-type>-<location-short>
  name_prefix = var.project_name
  # Strip hyphens for resources that disallow them (e.g., ACR, storage)
  name_alphanum = replace(var.project_name, "-", "")

  common_tags = merge(var.tags, {
    Project     = var.project_name
    ManagedBy   = "terraform"
    Environment = "production"
  })
}

# -----------------------------------------------------------------------------
# Resource Group
# -----------------------------------------------------------------------------

resource "azurerm_resource_group" "main" {
  name     = "rg-${local.name_prefix}"
  location = var.location
  tags     = local.common_tags
}

# -----------------------------------------------------------------------------
# Virtual Network + Subnets
# -----------------------------------------------------------------------------

resource "azurerm_virtual_network" "main" {
  name                = "vnet-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = ["10.0.0.0/16"]
  tags                = local.common_tags
}

resource "azurerm_subnet" "aks" {
  name                 = "snet-${local.name_prefix}-aks"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.0.0/20"]
}

resource "azurerm_subnet" "appgw" {
  name                 = "snet-${local.name_prefix}-appgw"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.16.0/24"]
}

# -----------------------------------------------------------------------------
# Log Analytics Workspace (conditional on enable_monitoring)
# -----------------------------------------------------------------------------

resource "azurerm_log_analytics_workspace" "main" {
  count = var.enable_monitoring ? 1 : 0

  name                = "law-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  tags                = local.common_tags
}

# -----------------------------------------------------------------------------
# User-Assigned Managed Identity for MicroFoundry Workload
# -----------------------------------------------------------------------------

resource "azurerm_user_assigned_identity" "microfoundry" {
  name                = "id-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.common_tags
}

# -----------------------------------------------------------------------------
# Azure Container Registry (ACR)
# -----------------------------------------------------------------------------

resource "azurerm_container_registry" "main" {
  name                = "acr${local.name_alphanum}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard"
  admin_enabled       = false
  tags                = local.common_tags
}

# -----------------------------------------------------------------------------
# Application Gateway (for AGIC)
# -----------------------------------------------------------------------------

resource "azurerm_public_ip" "appgw" {
  name                = "pip-${local.name_prefix}-appgw"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.common_tags
}

resource "azurerm_application_gateway" "main" {
  name                = "agw-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.common_tags

  sku {
    name     = "Standard_v2"
    tier     = "Standard_v2"
    capacity = 2
  }

  gateway_ip_configuration {
    name      = "appgw-ip-config"
    subnet_id = azurerm_subnet.appgw.id
  }

  frontend_ip_configuration {
    name                 = "appgw-frontend-ip"
    public_ip_address_id = azurerm_public_ip.appgw.id
  }

  frontend_port {
    name = "appgw-frontend-port-http"
    port = 80
  }

  frontend_port {
    name = "appgw-frontend-port-https"
    port = 443
  }

  # Default backend (placeholder -- AGIC manages actual backends)
  backend_address_pool {
    name = "appgw-default-backend"
  }

  backend_http_settings {
    name                  = "appgw-default-http-settings"
    cookie_based_affinity = "Disabled"
    port                  = 80
    protocol              = "Http"
    request_timeout       = 60
  }

  http_listener {
    name                           = "appgw-default-listener"
    frontend_ip_configuration_name = "appgw-frontend-ip"
    frontend_port_name             = "appgw-frontend-port-http"
    protocol                       = "Http"
  }

  request_routing_rule {
    name                       = "appgw-default-rule"
    priority                   = 100
    rule_type                  = "Basic"
    http_listener_name         = "appgw-default-listener"
    backend_address_pool_name  = "appgw-default-backend"
    backend_http_settings_name = "appgw-default-http-settings"
  }

  lifecycle {
    ignore_changes = [
      # AGIC manages these after initial creation
      backend_address_pool,
      backend_http_settings,
      http_listener,
      probe,
      request_routing_rule,
      redirect_configuration,
      url_path_map,
      frontend_port,
      tags["managed-by-k8s-ingress"],
    ]
  }
}

# -----------------------------------------------------------------------------
# AKS Cluster
# -----------------------------------------------------------------------------

resource "azurerm_kubernetes_cluster" "main" {
  name                = "aks-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = local.name_prefix
  kubernetes_version  = var.aks_kubernetes_version
  tags                = local.common_tags

  default_node_pool {
    name                = "system"
    node_count          = var.aks_node_count
    vm_size             = var.aks_vm_size
    vnet_subnet_id      = azurerm_subnet.aks.id
    os_disk_size_gb     = 128
    type                = "VirtualMachineScaleSets"
    enable_auto_scaling = false

    tags = local.common_tags
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin    = "azure"
    network_policy    = "calico"
    load_balancer_sku = "standard"
    service_cidr      = "10.1.0.0/16"
    dns_service_ip    = "10.1.0.10"
  }

  # Application Gateway Ingress Controller (AGIC) add-on
  ingress_application_gateway {
    gateway_id = azurerm_application_gateway.main.id
  }

  # Azure Monitor (OMS Agent) add-on
  dynamic "oms_agent" {
    for_each = var.enable_monitoring ? [1] : []
    content {
      log_analytics_workspace_id = azurerm_log_analytics_workspace.main[0].id
    }
  }

  # Enable OIDC issuer for Workload Identity
  oidc_issuer_enabled       = true
  workload_identity_enabled = true
}

# -----------------------------------------------------------------------------
# AKS-ACR Integration (Role Assignment)
# -----------------------------------------------------------------------------

resource "azurerm_role_assignment" "aks_acr_pull" {
  principal_id                     = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
  role_definition_name             = "AcrPull"
  scope                            = azurerm_container_registry.main.id
  skip_service_principal_aad_check = true
}

# -----------------------------------------------------------------------------
# Federated Identity Credential (Workload Identity for MicroFoundry SA)
# -----------------------------------------------------------------------------

resource "azurerm_federated_identity_credential" "microfoundry" {
  name                = "fid-${local.name_prefix}"
  resource_group_name = azurerm_resource_group.main.name
  parent_id           = azurerm_user_assigned_identity.microfoundry.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject             = "system:serviceaccount:microfoundry:microfoundry"
}

# -----------------------------------------------------------------------------
# Kubernetes Namespace
# -----------------------------------------------------------------------------

resource "kubernetes_namespace" "microfoundry" {
  metadata {
    name = "microfoundry"
    labels = {
      "app.kubernetes.io/managed-by" = "terraform"
      "app.kubernetes.io/part-of"    = var.project_name
    }
  }

  depends_on = [azurerm_kubernetes_cluster.main]
}

# -----------------------------------------------------------------------------
# Install MicroFoundry via Helm (OCI Registry)
# -----------------------------------------------------------------------------

resource "helm_release" "microfoundry" {
  name       = "microfoundry"
  namespace  = kubernetes_namespace.microfoundry.metadata[0].name
  repository = "oci://ghcr.io/younjinjeong/microfoundry/charts"
  chart      = "microfoundry"
  version    = var.mf_version
  wait       = true
  timeout    = 600

  values = [
    templatefile("${path.module}/../helm-values.yaml", {
      domain            = var.domain
      client_id         = azurerm_user_assigned_identity.microfoundry.client_id
      appgw_public_ip   = azurerm_public_ip.appgw.ip_address
    })
  ]

  depends_on = [
    azurerm_kubernetes_cluster.main,
    azurerm_role_assignment.aks_acr_pull,
    azurerm_federated_identity_credential.microfoundry,
  ]
}
