##############################################################################
# MicroFoundry — GCP GKE Deployment
# Main infrastructure: VPC, GKE, Artifact Registry, Workload Identity, Helm
##############################################################################

locals {
  cluster_name = "${var.project_name}-gke"
  network_name = "${var.project_name}-vpc"
  subnet_name  = "${var.project_name}-subnet"
  ar_repo_name = "${var.project_name}-images"
  sa_name      = "${var.project_name}-workload"
  k8s_sa_name  = "microfoundry"
  namespace    = "microfoundry"

  # Merge user-provided tags with mandatory labels
  labels = merge(var.tags, {
    managed-by = "terraform"
    project    = var.project_name
  })
}

# ---------------------------------------------------------------------------
# Provider configuration
# ---------------------------------------------------------------------------

provider "google" {
  project = var.project_id
  region  = var.region
}

# The kubernetes and helm providers are configured after the GKE cluster
# is created so we can use its endpoint and credentials.
provider "kubernetes" {
  host                   = "https://${google_container_cluster.this.endpoint}"
  token                  = data.google_client_config.current.access_token
  cluster_ca_certificate = base64decode(google_container_cluster.this.master_auth[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = "https://${google_container_cluster.this.endpoint}"
    token                  = data.google_client_config.current.access_token
    cluster_ca_certificate = base64decode(google_container_cluster.this.master_auth[0].cluster_ca_certificate)
  }
}

data "google_client_config" "current" {}

# ---------------------------------------------------------------------------
# Enable required GCP APIs
# ---------------------------------------------------------------------------

resource "google_project_service" "apis" {
  for_each = toset([
    "container.googleapis.com",
    "artifactregistry.googleapis.com",
    "compute.googleapis.com",
    "iam.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# ---------------------------------------------------------------------------
# VPC Network
# ---------------------------------------------------------------------------

resource "google_compute_network" "this" {
  name                    = local.network_name
  project                 = var.project_id
  auto_create_subnetworks = false

  depends_on = [google_project_service.apis]
}

resource "google_compute_subnetwork" "this" {
  name          = local.subnet_name
  project       = var.project_id
  region        = var.region
  network       = google_compute_network.this.id
  ip_cidr_range = "10.0.0.0/20"

  secondary_ip_range {
    range_name    = "${local.subnet_name}-pods"
    ip_cidr_range = "10.4.0.0/14"
  }

  secondary_ip_range {
    range_name    = "${local.subnet_name}-services"
    ip_cidr_range = "10.8.0.0/20"
  }

  private_ip_google_access = true
}

# Cloud NAT so private nodes can pull images from the internet
resource "google_compute_router" "this" {
  name    = "${var.project_name}-router"
  project = var.project_id
  region  = var.region
  network = google_compute_network.this.id
}

resource "google_compute_router_nat" "this" {
  name                               = "${var.project_name}-nat"
  project                            = var.project_id
  region                             = var.region
  router                             = google_compute_router.this.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# ---------------------------------------------------------------------------
# GKE Cluster
# ---------------------------------------------------------------------------

resource "google_container_cluster" "this" {
  name     = local.cluster_name
  project  = var.project_id
  location = var.region

  # Network configuration
  network    = google_compute_network.this.id
  subnetwork = google_compute_subnetwork.this.id

  ip_allocation_policy {
    cluster_secondary_range_name  = "${local.subnet_name}-pods"
    services_secondary_range_name = "${local.subnet_name}-services"
  }

  # Autopilot vs Standard
  dynamic "cluster_autoscaling" {
    for_each = var.use_autopilot ? [] : [1]
    content {
      enabled = false
    }
  }

  enable_autopilot = var.use_autopilot

  # When using Standard mode we manage the node pool separately.
  # Remove the default node pool that GKE creates.
  dynamic "node_pool" {
    for_each = var.use_autopilot ? [] : [1]
    content {
      name               = "default-pool"
      initial_node_count = 1
    }
  }
  remove_default_node_pool = var.use_autopilot ? false : true

  # Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Monitoring and Logging
  logging_service    = var.enable_monitoring ? "logging.googleapis.com/kubernetes" : "none"
  monitoring_service = var.enable_monitoring ? "monitoring.googleapis.com/kubernetes" : "none"

  # Release channel
  release_channel {
    channel = "REGULAR"
  }

  # Private cluster: nodes have internal IPs only
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  # Master authorized networks — allow all by default; restrict for production
  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "0.0.0.0/0"
      display_name = "All networks"
    }
  }

  resource_labels = local.labels

  deletion_protection = false

  depends_on = [
    google_project_service.apis,
    google_compute_subnetwork.this,
  ]
}

# ---------------------------------------------------------------------------
# GKE Standard Node Pool (only when not using Autopilot)
# ---------------------------------------------------------------------------

resource "google_container_node_pool" "primary" {
  count = var.use_autopilot ? 0 : 1

  name     = "${var.project_name}-primary"
  project  = var.project_id
  location = var.region
  cluster  = google_container_cluster.this.name

  node_count = var.gke_node_count

  node_config {
    machine_type    = var.gke_machine_type
    service_account = google_service_account.gke_nodes.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    labels = local.labels

    metadata = {
      disable-legacy-endpoints = "true"
    }
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# Service account for GKE nodes
resource "google_service_account" "gke_nodes" {
  account_id   = "${var.project_name}-gke-nodes"
  display_name = "GKE Node Service Account for ${var.project_name}"
  project      = var.project_id
}

resource "google_project_iam_member" "gke_nodes_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

resource "google_project_iam_member" "gke_nodes_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

resource "google_project_iam_member" "gke_nodes_ar_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

# ---------------------------------------------------------------------------
# Artifact Registry
# ---------------------------------------------------------------------------

resource "google_artifact_registry_repository" "this" {
  repository_id = local.ar_repo_name
  project       = var.project_id
  location      = var.region
  format        = "DOCKER"
  description   = "Container images for ${var.project_name}"

  labels = local.labels

  depends_on = [google_project_service.apis]
}

# ---------------------------------------------------------------------------
# Workload Identity — GCP SA bound to K8s SA
# ---------------------------------------------------------------------------

resource "google_service_account" "workload" {
  account_id   = local.sa_name
  display_name = "MicroFoundry Workload Identity SA"
  project      = var.project_id
}

# Grant the GCP SA permissions it needs
resource "google_project_iam_member" "workload_ar_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.workload.email}"
}

resource "google_project_iam_member" "workload_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.workload.email}"
}

resource "google_project_iam_member" "workload_monitoring_viewer" {
  project = var.project_id
  role    = "roles/monitoring.viewer"
  member  = "serviceAccount:${google_service_account.workload.email}"
}

# Bind K8s SA -> GCP SA via Workload Identity
resource "google_service_account_iam_member" "workload_identity_binding" {
  service_account_id = google_service_account.workload.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${local.namespace}/${local.k8s_sa_name}]"
}

# ---------------------------------------------------------------------------
# Static external IP for the HTTP(S) Load Balancer
# ---------------------------------------------------------------------------

resource "google_compute_global_address" "ingress" {
  name    = "${var.project_name}-ingress-ip"
  project = var.project_id
}

# ---------------------------------------------------------------------------
# Install MicroFoundry via Helm
# ---------------------------------------------------------------------------

resource "helm_release" "microfoundry" {
  name             = "microfoundry"
  namespace        = local.namespace
  create_namespace = true

  repository = "oci://ghcr.io/younjinjeong/microfoundry/charts"
  chart      = "microfoundry"
  version    = var.mf_version

  values = [
    templatefile("${path.module}/../helm-values.yaml", {
      project_id       = var.project_id
      gcp_sa_email     = google_service_account.workload.email
      domain           = var.domain
      static_ip_name   = google_compute_global_address.ingress.name
      enable_monitoring = var.enable_monitoring
    })
  ]

  depends_on = [
    google_container_cluster.this,
    google_container_node_pool.primary,
    google_service_account_iam_member.workload_identity_binding,
  ]
}
