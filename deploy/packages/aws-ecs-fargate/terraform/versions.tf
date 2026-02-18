# -----------------------------------------------------------------------------
# MicroFoundry AWS ECS Fargate Deployment — Provider and Terraform Versions
# -----------------------------------------------------------------------------
# This configuration defines the required Terraform version and AWS provider
# for the hybrid deployment: ECS Fargate (control plane) + EKS (workloads).
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region

  default_tags {
    tags = merge(var.tags, {
      Project   = var.project_name
      ManagedBy = "terraform"
    })
  }
}
