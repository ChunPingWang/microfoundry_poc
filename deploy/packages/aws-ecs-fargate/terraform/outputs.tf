# -----------------------------------------------------------------------------
# MicroFoundry AWS ECS Fargate Deployment — Outputs
# -----------------------------------------------------------------------------

output "alb_dns_name" {
  description = "DNS name of the ALB fronting the MicroFoundry admin UI."
  value       = aws_lb.mf.dns_name
}

output "alb_url" {
  description = "Full HTTP URL to the MicroFoundry admin UI."
  value       = "http://${aws_lb.mf.dns_name}"
}

output "eks_cluster_name" {
  description = "Name of the EKS cluster used for application workloads."
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "API endpoint of the EKS workload cluster."
  value       = module.eks.cluster_endpoint
}

output "ecr_repository_url" {
  description = "URL of the ECR repository for MicroFoundry container images."
  value       = aws_ecr_repository.mf.repository_url
}

output "ecs_cluster_name" {
  description = "Name of the ECS cluster running the MicroFoundry control plane."
  value       = aws_ecs_cluster.mf.name
}

output "kubeconfig_command" {
  description = "AWS CLI command to configure kubectl for the EKS workload cluster."
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}

output "efs_file_system_id" {
  description = "EFS file system ID used for MicroFoundry persistent configuration."
  value       = aws_efs_file_system.mf_config.id
}

output "vpc_id" {
  description = "VPC ID housing all MicroFoundry resources."
  value       = module.vpc.vpc_id
}
