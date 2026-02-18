# MicroFoundry on GCP GKE

Deploy MicroFoundry to a Google Kubernetes Engine (GKE) cluster with Terraform.

## Architecture

This package provisions the following GCP resources:

- **VPC Network** with a private subnet, secondary ranges for pods/services, and Cloud NAT
- **GKE Cluster** (Standard or Autopilot) with Workload Identity enabled
- **Artifact Registry** Docker repository for container images
- **Workload Identity** binding between a GCP service account and the MicroFoundry Kubernetes service account
- **Global static IP** for the GCP HTTP(S) Load Balancer (GCE Ingress)
- **Cloud Monitoring and Cloud Logging** enabled by default (replaces self-hosted Prometheus/Grafana/Loki)
- **MicroFoundry** installed via the Helm chart from the OCI registry

## Prerequisites

| Tool | Minimum Version | Install |
|------|----------------|---------|
| Terraform | >= 1.5 | https://developer.hashicorp.com/terraform/install |
| gcloud CLI | latest | https://cloud.google.com/sdk/docs/install |
| kubectl | >= 1.27 | `gcloud components install kubectl` |
| Helm | >= 3.12 | https://helm.sh/docs/intro/install/ |

You must be authenticated with gcloud:

```bash
gcloud auth login
gcloud config set project <YOUR_PROJECT_ID>
```

## Quick Start

### 1. Configure variables

Create a `terraform.tfvars` file in the `terraform/` directory:

```hcl
project_id       = "my-gcp-project"
project_name     = "microfoundry"
region           = "us-central1"
domain           = "apps.example.com"
mf_version       = "0.1.0"
gke_machine_type = "e2-medium"
gke_node_count   = 2
enable_monitoring = true
```

To use GKE Autopilot instead of Standard mode:

```hcl
use_autopilot = true
```

### 2. Run the install script

```bash
chmod +x install.sh
./install.sh
```

The script will:
1. Check that all prerequisites are installed
2. Initialize and apply Terraform
3. Configure kubectl with the new cluster credentials
4. Print access information

### 3. (Alternative) Manual Terraform workflow

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

Then configure kubectl:

```bash
gcloud container clusters get-credentials microfoundry-gke \
  --region us-central1 \
  --project my-gcp-project
```

## DNS Configuration

After deployment, point your DNS records to the static IP output by Terraform:

```
A  admin.apps.example.com  -> <INGRESS_IP>
A  *.apps.example.com      -> <INGRESS_IP>
```

Retrieve the IP address:

```bash
terraform -chdir=terraform output ingress_ip
```

## TLS (HTTPS)

To enable TLS with a Google-managed certificate, add a `ManagedCertificate` resource
to your cluster and reference it in the Ingress annotations. Example:

```yaml
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: microfoundry-cert
  namespace: microfoundry
spec:
  domains:
    - admin.apps.example.com
    - "*.apps.example.com"
```

Then add the annotation to the Ingress (update `helm-values.yaml`):

```yaml
ingress:
  annotations:
    networking.gke.io/managed-certificates: microfoundry-cert
```

## Outputs

| Output | Description |
|--------|-------------|
| `kubeconfig_command` | gcloud command to configure kubectl |
| `gke_cluster_name` | Name of the GKE cluster |
| `gke_cluster_endpoint` | GKE API server endpoint (sensitive) |
| `artifact_registry_url` | Artifact Registry Docker repository URL |
| `mf_admin_url` | MicroFoundry Admin UI URL |
| `ingress_ip` | Static external IP for the load balancer |
| `workload_identity_sa` | GCP service account bound via Workload Identity |

## MicroFoundry Configuration

Copy `mf.example.yaml` to `~/.mf/mf.yaml` and update the values for your
environment. The key differences from a local deployment:

- `kubernetes.active` is set to `"gke"`
- `provider` is `"gke"` instead of `"docker-desktop"` or `"native"`
- Monitoring URLs are empty because GKE uses Cloud Monitoring/Logging
- `beyla_enabled` is `false` (eBPF auto-instrumentation is not needed on GKE)

## Tear Down

To destroy all provisioned resources:

```bash
terraform -chdir=terraform destroy
```

This will remove the GKE cluster, VPC, Artifact Registry, IAM bindings, and all
other resources created by this package.

## File Structure

```
gcp-gke/
  terraform/
    versions.tf      Terraform and provider version constraints
    variables.tf     Input variables
    main.tf          VPC, GKE, Artifact Registry, Workload Identity, Helm
    outputs.tf       Output values
  helm-values.yaml   Helm values template for GKE (consumed by Terraform)
  mf.example.yaml    Example MicroFoundry config for GKE
  install.sh         One-command installation script
  README.md          This file
```
