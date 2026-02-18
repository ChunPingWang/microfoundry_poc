##############################################################################
# MicroFoundry — GCP GKE Deployment
# Input variables
##############################################################################

variable "project_id" {
  description = "GCP project ID"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id must be a valid GCP project ID (6-30 chars, lowercase letters, digits, hyphens)."
  }
}

variable "project_name" {
  description = "Name prefix for all resources"
  type        = string
  default     = "microfoundry"
}

variable "region" {
  description = "GCP region for all resources"
  type        = string
  default     = "us-central1"
}

variable "domain" {
  description = "Base domain for MicroFoundry applications (e.g., apps.example.com)"
  type        = string
  default     = ""
}

variable "mf_version" {
  description = "MicroFoundry Helm chart / container image version"
  type        = string
  default     = "0.1.0"
}

variable "gke_machine_type" {
  description = "GCE machine type for GKE node pool (ignored when use_autopilot is true)"
  type        = string
  default     = "e2-medium"
}

variable "gke_node_count" {
  description = "Number of nodes per zone in the default node pool (ignored when use_autopilot is true)"
  type        = number
  default     = 2

  validation {
    condition     = var.gke_node_count >= 1 && var.gke_node_count <= 100
    error_message = "gke_node_count must be between 1 and 100."
  }
}

variable "use_autopilot" {
  description = "Use GKE Autopilot instead of Standard mode"
  type        = bool
  default     = false
}

variable "enable_monitoring" {
  description = "Enable GCP Cloud Monitoring and Cloud Logging integration on the GKE cluster"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Additional tags to apply to all resources as labels"
  type        = map(string)
  default     = {}
}
