# MicroFoundry - AWS EKS Deployment Package

Deploy MicroFoundry on Amazon Elastic Kubernetes Service (EKS) with production-grade
infrastructure managed by Terraform.

## Prerequisites

| Tool      | Version   | Purpose                       |
|-----------|-----------|-------------------------------|
| Terraform | >= 1.5    | Infrastructure provisioning   |
| AWS CLI   | v2        | AWS authentication and ECR    |
| kubectl   | >= 1.28   | Kubernetes cluster management |
| Helm      | >= 3.0    | Chart installation            |

You will also need:

- An AWS account with permissions to create VPC, EKS, ECR, IAM, and CloudWatch resources
- AWS credentials configured (`aws configure` or environment variables)
- A registered domain name (for ALB ingress routing)

## Architecture

```
                         Internet
                            |
                    +-------+-------+
                    |  Route 53 /   |
                    |  External DNS |
                    +-------+-------+
                            |
                    +-------+-------+
                    | AWS ALB       |
                    | (Load         |
                    |  Balancer     |
                    |  Controller)  |
                    +-------+-------+
                            |
              +-------------+-------------+
              |        AWS VPC            |
              |  10.0.0.0/16              |
              |                           |
              |  +-----+ +-----+ +-----+ |
              |  |AZ-a | |AZ-b | |AZ-c | |
              |  |pub  | |pub  | |pub  | |  <-- Public subnets (ALB)
              |  +-----+ +-----+ +-----+ |
              |  +-----+ +-----+ +-----+ |
              |  |AZ-a | |AZ-b | |AZ-c | |  <-- Private subnets (EKS nodes)
              |  |priv | |priv | |priv | |
              |  +-----+ +-----+ +-----+ |
              |         |                 |
              |  +------+------+          |
              |  | EKS Cluster |          |
              |  |             |          |
              |  | +---------+ |          |
              |  | | Managed | |          |
              |  | | Node    | |          |
              |  | | Group   | |          |
              |  | +---------+ |          |
              |  |             |          |
              |  | MicroFoundry|          |
              |  | (Helm)      |          |
              |  +-------------+          |
              |                           |
              |  +-----+   +-----------+  |
              |  | ECR |   | CloudWatch|  |
              |  |     |   | Container |  |
              |  |     |   | Insights  |  |
              |  +-----+   +-----------+  |
              +---------------------------+
```

## Quick Install

1. Clone this repository and navigate to the package directory:

   ```bash
   cd microfoundry/deploy/packages/aws-eks
   ```

2. Create a `terraform.tfvars` file in the `terraform/` directory:

   ```hcl
   project_name         = "microfoundry"
   region               = "us-east-1"
   domain               = "apps.example.com"
   mf_version           = "0.1.0"
   eks_node_instance_type = "t3.medium"
   eks_node_count       = 2
   enable_monitoring    = true

   tags = {
     Team        = "platform"
     Environment = "production"
   }
   ```

3. Run the installation script:

   ```bash
   chmod +x install.sh
   ./install.sh
   ```

   Or run Terraform manually:

   ```bash
   cd terraform
   terraform init
   terraform plan
   terraform apply
   ```

4. Configure kubectl:

   ```bash
   aws eks update-kubeconfig --region us-east-1 --name microfoundry-eks
   ```

5. Verify the deployment:

   ```bash
   kubectl get pods -n microfoundry
   kubectl get ingress -n microfoundry
   ```

## What Gets Provisioned

### VPC (terraform-aws-modules/vpc/aws)

- CIDR: `10.0.0.0/16`
- 3 public subnets (one per AZ) for the ALB
- 3 private subnets (one per AZ) for EKS worker nodes
- Single NAT gateway for outbound internet access from private subnets
- DNS hostnames and DNS support enabled

### EKS Cluster (terraform-aws-modules/eks/aws)

- Managed Kubernetes control plane (version configurable, default 1.29)
- Managed node group with configurable instance type and count
- IRSA (IAM Roles for Service Accounts) via OIDC provider
- CoreDNS, kube-proxy, and VPC CNI add-ons
- Public API endpoint for cluster management

### ECR (Elastic Container Registry)

- Private repository for application images (`<project>/app`)
- Image scanning on push enabled
- Lifecycle policy retains the last 30 images

### AWS Load Balancer Controller

- Installed via Helm into `kube-system`
- IRSA-bound service account with required IAM permissions
- Routes traffic from internet-facing ALBs to cluster services
- Ingress class: `alb`

### CloudWatch Container Insights (optional)

- Enabled by default (`enable_monitoring = true`)
- Provides CPU, memory, disk, and network metrics for nodes and pods
- Container-level log aggregation to CloudWatch Logs
- Set `enable_monitoring = false` to disable

## CSP-Native Services

This deployment package uses AWS-managed services instead of self-hosted
equivalents wherever possible:

| Capability        | Self-Hosted (local)          | AWS EKS (this package)         |
|-------------------|------------------------------|--------------------------------|
| Kubernetes        | kind / minikube              | Amazon EKS                     |
| Container Registry| N/A                          | Amazon ECR                     |
| Load Balancing    | NGINX Ingress                | AWS ALB (Load Balancer Ctrl)   |
| Metrics           | Prometheus                   | CloudWatch Container Insights  |
| Logs              | Loki                         | CloudWatch Logs                |
| Dashboards        | Grafana                      | CloudWatch Dashboards          |
| Alerts            | Alertmanager                 | CloudWatch Alarms              |
| Auto-instrumentation | Beyla                     | CloudWatch (disabled: beyla)   |

The Helm values in this package set all self-hosted monitoring URLs to empty
strings and disable Beyla, since CloudWatch Container Insights provides the
equivalent functionality natively.

## Configuration

### Terraform Variables

| Variable                | Default          | Description                              |
|-------------------------|------------------|------------------------------------------|
| `project_name`          | `microfoundry`   | Name prefix for all resources            |
| `region`                | `us-east-1`      | AWS region                               |
| `domain`                | (required)       | Base domain for applications             |
| `mf_version`            | (required)       | MicroFoundry Helm chart version          |
| `eks_node_instance_type`| `t3.medium`      | EC2 instance type for worker nodes       |
| `eks_node_count`        | `2`              | Desired number of worker nodes           |
| `eks_cluster_version`   | `1.29`           | Kubernetes version                       |
| `enable_monitoring`     | `true`           | Enable CloudWatch Container Insights     |
| `tags`                  | `{}`             | Additional tags for all resources        |

### Helm Values

The `helm-values.yaml` file is a Terraform template. It is automatically
populated with values from Terraform outputs (IAM role ARN, domain, cluster
name). To use it outside of Terraform, replace the `${...}` placeholders
manually.

### MicroFoundry Config

Copy `mf.example.yaml` to customize the MicroFoundry application
configuration. The file includes a commented-out authentication section
with a Keycloak OIDC example.

## Cost Estimates

Approximate monthly costs (us-east-1, on-demand pricing):

| Resource                     | Specification      | Estimated Cost   |
|------------------------------|--------------------|------------------|
| EKS control plane            | 1 cluster          | ~$73/month       |
| EC2 instances (t3.medium x2) | 2 nodes            | ~$61/month       |
| NAT Gateway                  | 1 gateway + data   | ~$32/month       |
| ALB                          | 1 load balancer    | ~$16/month       |
| ECR                          | Storage + transfer | ~$1/month        |
| CloudWatch                   | Metrics + logs     | ~$10-30/month    |
| **Total**                    |                    | **~$193-213/month** |

Costs vary by region, data transfer volume, and usage patterns. Use the
[AWS Pricing Calculator](https://calculator.aws/) for precise estimates.
Reserved instances or Savings Plans can reduce EC2 costs by 30-60%.

## Cleanup

To destroy all resources created by this package:

```bash
cd terraform
terraform destroy
```

This will remove the EKS cluster, VPC, ECR repository, IAM roles, and all
associated resources. Ensure you have backed up any container images in ECR
and any persistent data before running destroy.

## Troubleshooting

### ALB not provisioning

Verify the AWS Load Balancer Controller is running:

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller
```

Check controller logs:

```bash
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller
```

### Nodes not joining the cluster

Check the managed node group status:

```bash
aws eks describe-nodegroup --cluster-name microfoundry-eks --nodegroup-name microfoundry-nodes
```

### IRSA not working

Verify the service account annotation:

```bash
kubectl get sa microfoundry -n microfoundry -o yaml
```

The annotation `eks.amazonaws.com/role-arn` should contain the IAM role ARN.

### Terraform state

This package uses local Terraform state by default. For team environments,
configure a remote backend (S3 + DynamoDB) in `versions.tf`:

```hcl
terraform {
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "microfoundry/eks/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"
    encrypt        = true
  }
}
```
