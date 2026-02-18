# -----------------------------------------------------------------------------
# MicroFoundry - Azure AKS Deployment Package
# Input Variables
# -----------------------------------------------------------------------------

variable "project_name" {
  description = "Name prefix for all resources created by this module"
  type        = string
  default     = "microfoundry"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,28}[a-z0-9]$", var.project_name))
    error_message = "Project name must be 3-30 lowercase alphanumeric characters or hyphens, starting with a letter."
  }
}

variable "location" {
  description = "Azure region for all resources"
  type        = string
  default     = "eastus"
}

variable "domain" {
  description = "Base domain for applications (e.g., apps.example.com)"
  type        = string
}

variable "mf_version" {
  description = "MicroFoundry version to deploy (Helm chart tag)"
  type        = string
}

variable "aks_vm_size" {
  description = "VM size for the AKS system node pool"
  type        = string
  default     = "Standard_D2s_v3"
}

variable "aks_node_count" {
  description = "Number of nodes in the AKS system node pool"
  type        = number
  default     = 2

  validation {
    condition     = var.aks_node_count >= 1 && var.aks_node_count <= 20
    error_message = "Node count must be between 1 and 20."
  }
}

variable "aks_kubernetes_version" {
  description = "Kubernetes version for the AKS cluster"
  type        = string
  default     = "1.29"
}

variable "enable_monitoring" {
  description = "Enable Azure Monitor and Log Analytics for cluster observability"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}
