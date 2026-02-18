# -----------------------------------------------------------------------------
# MicroFoundry - AWS EKS Deployment Package
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

variable "region" {
  description = "AWS region for all resources"
  type        = string
  default     = "us-east-1"
}

variable "domain" {
  description = "Base domain for applications (e.g., apps.example.com)"
  type        = string
}

variable "mf_version" {
  description = "MicroFoundry version to deploy (Helm chart tag)"
  type        = string
}

variable "eks_node_instance_type" {
  description = "EC2 instance type for EKS managed node group"
  type        = string
  default     = "t3.medium"
}

variable "eks_node_count" {
  description = "Desired number of nodes in the EKS managed node group"
  type        = number
  default     = 2

  validation {
    condition     = var.eks_node_count >= 1 && var.eks_node_count <= 20
    error_message = "Node count must be between 1 and 20."
  }
}

variable "eks_cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.29"
}

variable "enable_monitoring" {
  description = "Enable CloudWatch Container Insights for cluster monitoring"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}
