# -----------------------------------------------------------------------------
# MicroFoundry - AWS EKS Deployment Package
# Outputs
# -----------------------------------------------------------------------------

output "kubeconfig_command" {
  description = "Command to configure kubectl for this EKS cluster"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}

output "eks_cluster_name" {
  description = "Name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint URL for the EKS cluster API server"
  value       = module.eks.cluster_endpoint
}

output "ecr_repository_url" {
  description = "URL of the ECR repository for application images"
  value       = aws_ecr_repository.app.repository_url
}

output "mf_admin_url" {
  description = "URL to access the MicroFoundry admin UI"
  value       = "https://admin.${var.domain}"
}

output "helm_install_command" {
  description = "Helm command to manually install or upgrade MicroFoundry"
  value       = "helm upgrade --install microfoundry oci://ghcr.io/younjinjeong/microfoundry/charts/microfoundry --version ${var.mf_version} --namespace microfoundry -f ../helm-values.yaml"
}

# -----------------------------------------------------------------------------
# Additional Outputs (useful for debugging and integration)
# -----------------------------------------------------------------------------

output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "IDs of the private subnets"
  value       = module.vpc.private_subnets
}

output "microfoundry_iam_role_arn" {
  description = "ARN of the IAM role for the MicroFoundry service account (IRSA)"
  value       = aws_iam_role.microfoundry.arn
}

output "cluster_oidc_provider" {
  description = "OIDC provider URL for the EKS cluster (used for IRSA)"
  value       = module.eks.oidc_provider
}
