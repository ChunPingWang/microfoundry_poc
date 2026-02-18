# -----------------------------------------------------------------------------
# MicroFoundry - Azure AKS Deployment Package
# Outputs
# -----------------------------------------------------------------------------

output "kubeconfig_command" {
  description = "Command to configure kubectl for this AKS cluster"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_kubernetes_cluster.main.name} --overwrite-existing"
}

output "aks_cluster_name" {
  description = "Name of the AKS cluster"
  value       = azurerm_kubernetes_cluster.main.name
}

output "aks_resource_group" {
  description = "Name of the resource group containing all resources"
  value       = azurerm_resource_group.main.name
}

output "acr_login_server" {
  description = "Login server URL for the Azure Container Registry"
  value       = azurerm_container_registry.main.login_server
}

output "mf_admin_url" {
  description = "MicroFoundry admin dashboard URL"
  value       = "https://admin.${var.domain}"
}

output "appgw_public_ip" {
  description = "Public IP address of the Application Gateway"
  value       = azurerm_public_ip.appgw.ip_address
}

output "managed_identity_client_id" {
  description = "Client ID of the user-assigned managed identity for MicroFoundry"
  value       = azurerm_user_assigned_identity.microfoundry.client_id
}

output "log_analytics_workspace_id" {
  description = "Log Analytics workspace ID (empty if monitoring is disabled)"
  value       = var.enable_monitoring ? azurerm_log_analytics_workspace.main[0].id : ""
}
