##############################################################################
# MicroFoundry — GCP GKE Deployment
# Outputs
##############################################################################

output "kubeconfig_command" {
  description = "gcloud command to configure kubectl for this cluster"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.this.name} --region ${var.region} --project ${var.project_id}"
}

output "gke_cluster_name" {
  description = "Name of the GKE cluster"
  value       = google_container_cluster.this.name
}

output "gke_cluster_endpoint" {
  description = "GKE cluster API server endpoint"
  value       = google_container_cluster.this.endpoint
  sensitive   = true
}

output "artifact_registry_url" {
  description = "Artifact Registry Docker repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.this.repository_id}"
}

output "mf_admin_url" {
  description = "MicroFoundry Admin UI URL"
  value       = var.domain != "" ? "https://admin.${var.domain}" : "http://${google_compute_global_address.ingress.address}"
}

output "ingress_ip" {
  description = "Static external IP address allocated for the HTTP(S) Load Balancer"
  value       = google_compute_global_address.ingress.address
}

output "workload_identity_sa" {
  description = "GCP service account email bound to the MicroFoundry K8s service account via Workload Identity"
  value       = google_service_account.workload.email
}
