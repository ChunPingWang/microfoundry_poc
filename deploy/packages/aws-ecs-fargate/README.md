# MicroFoundry -- AWS ECS Fargate + EKS Deployment

This package deploys MicroFoundry as a **hybrid architecture** on AWS:

- **Control Plane** (MicroFoundry admin) runs on **ECS Fargate** -- serverless, no EC2 instances to manage.
- **Application Workloads** run on **EKS** -- a managed Kubernetes cluster that MicroFoundry orchestrates.

```
                    +------------------+
                    |    Internet      |
                    +--------+---------+
                             |
                    +--------v---------+
                    | ALB (HTTP/HTTPS) |
                    +--------+---------+
                             |
               +-------------v--------------+
               | ECS Fargate                 |
               | +------------------------+  |
               | | mf-admin container     |  |
               | | (MicroFoundry UI:8080) |  |
               | +----------+-------------+  |
               |            |                |
               |   +--------v---------+     |
               |   | EFS (mf.yaml)    |     |
               +---|-------------------+-----+
                   |
          +--------v---------+
          | EKS Cluster      |
          | +------+ +-----+ |
          | | app1 | | app2| |
          | +------+ +-----+ |
          | (workloads)       |
          +-------------------+
```

## Prerequisites

| Tool        | Version   | Purpose                            |
|-------------|-----------|-------------------------------------|
| Terraform   | >= 1.5    | Infrastructure provisioning         |
| AWS CLI     | >= 2.x    | AWS authentication and ECR access   |
| Docker      | >= 20.x   | Building and pushing container images |
| kubectl     | >= 1.28   | (Optional) Direct EKS access        |

You also need:

- An AWS account with permissions to create VPC, ECS, EKS, ALB, ECR, EFS, IAM, and CloudWatch resources.
- AWS credentials configured (`aws configure` or environment variables).
- A domain name for your applications (configured in `terraform.tfvars`).

## Quick Install

### 1. Configure Variables

Create `terraform/terraform.tfvars`:

```hcl
domain     = "apps.example.com"
mf_version = "0.1.0"
region     = "us-east-1"

# Optional overrides:
# project_name            = "microfoundry"
# fargate_cpu             = 512
# fargate_memory          = 1024
# eks_node_instance_type  = "t3.medium"
# eks_node_count          = 2
# eks_cluster_version     = "1.29"
# tags = {
#   Environment = "production"
#   Team        = "platform"
# }
```

### 2. Push the MicroFoundry Image to ECR

After the first `terraform apply` creates the ECR repository, push your image:

```bash
# Authenticate Docker to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

# Tag and push
docker tag mf:0.1.0 <account-id>.dkr.ecr.us-east-1.amazonaws.com/microfoundry/mf:0.1.0
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/microfoundry/mf:0.1.0
```

### 3. Run the Installer

```bash
chmod +x install.sh
./install.sh
```

Or run Terraform directly:

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

The script validates prerequisites, runs `terraform init`, `plan`, and `apply`, then prints the ALB URL and cluster details.

### 4. Configure MicroFoundry

Copy `mf.example.yaml` to the EFS volume as `mf.yaml` and edit it with your domain and cluster details. The EFS volume is mounted into the Fargate container at `/etc/microfoundry`.

## What Gets Provisioned

| Resource                        | Purpose                                         |
|---------------------------------|-------------------------------------------------|
| **VPC**                         | Isolated network with 3 AZs, public + private subnets, NAT gateway |
| **ECS Cluster (Fargate)**       | Serverless compute for the MicroFoundry admin control plane |
| **ECS Task Definition**         | Container config: image, port 8080, EFS mount, health check |
| **ECS Service**                 | Maintains desired task count, wired to ALB target group |
| **ALB**                         | Internet-facing load balancer routing traffic to Fargate |
| **EKS Cluster**                 | Managed Kubernetes cluster for application workloads |
| **EKS Managed Node Group**      | EC2 worker nodes (t3.medium x 2 by default)     |
| **ECR Repository**              | Container registry for the MicroFoundry image    |
| **EFS File System**             | Persistent storage for MicroFoundry configuration |
| **CloudWatch Log Group**        | Centralized Fargate container logs (`/ecs/microfoundry`) |
| **IAM Roles**                   | Task execution role (ECR/logs), task role (EKS/ECR/EFS access) |
| **Security Groups**             | ALB (HTTP/HTTPS), Fargate (8080 from ALB), EFS (NFS from Fargate), EKS (443 from Fargate) |

## How Fargate Accesses EKS

MicroFoundry on Fargate manages Kubernetes workloads on the EKS cluster through IAM role mapping:

1. **ECS Task Role** (`microfoundry-ecs-task`) is assigned to the Fargate task. This role has `eks:DescribeCluster`, `eks:AccessKubernetesApi`, and ECR permissions.

2. **EKS Access Entry** maps the ECS task role as a cluster admin on the EKS cluster using the `AmazonEKSClusterAdminPolicy`. This grants the Fargate container full Kubernetes API access.

3. **At runtime**, the MicroFoundry process inside the Fargate container uses the AWS SDK to discover the EKS cluster endpoint and obtain a temporary authentication token via the task role. No static kubeconfig or long-lived credentials are stored.

This approach follows AWS best practices:
- No hardcoded credentials -- IAM role-based authentication
- Least-privilege scoping -- the task role can only access the specific EKS cluster
- Automatic credential rotation via STS temporary tokens

## CloudWatch Monitoring

All Fargate container logs are shipped to CloudWatch Logs automatically.

**View logs:**

```bash
# Tail logs in real time
aws logs tail /ecs/microfoundry --follow

# Search logs
aws logs filter-log-events \
  --log-group-name /ecs/microfoundry \
  --filter-pattern "ERROR"
```

**Container Insights** is enabled on the ECS cluster, providing:
- CPU and memory utilization metrics
- Running task count
- Service-level metrics in the CloudWatch console

For workload-level monitoring on EKS, deploy a Prometheus + Grafana stack on the EKS cluster and configure the monitoring URLs in `mf.yaml`.

## Cost Estimates

Approximate monthly costs for the default configuration in us-east-1 (as of 2025):

| Resource                  | Configuration              | Estimated Cost   |
|---------------------------|----------------------------|------------------|
| ECS Fargate               | 0.5 vCPU, 1 GB memory     | ~$15/month       |
| EKS Control Plane         | Managed cluster            | ~$73/month       |
| EKS Worker Nodes          | 2x t3.medium               | ~$60/month       |
| NAT Gateway               | Single AZ                  | ~$32/month       |
| ALB                       | Application Load Balancer  | ~$16/month       |
| EFS                       | Minimal usage (<1 GB)      | ~$0.30/month     |
| CloudWatch Logs           | ~1 GB/month                | ~$0.50/month     |
| ECR                       | ~1 GB storage              | ~$0.10/month     |
| **Total**                 |                            | **~$197/month**  |

These are estimates. Actual costs depend on traffic, log volume, and data transfer. Use the [AWS Pricing Calculator](https://calculator.aws/) for precise figures.

To reduce costs:
- Use `FARGATE_SPOT` capacity provider for non-production environments
- Reduce `eks_node_count` to 1 for development
- Use smaller instance types (e.g., `t3.small`)

## Cleanup

To destroy all resources:

```bash
cd terraform
terraform destroy
```

This removes all AWS resources created by this deployment. The ECR repository is configured with `force_delete = true`, so it will be removed even if it contains images.

**Important**: Verify that no critical workloads are running on the EKS cluster before destroying the infrastructure. Terraform will remove the EKS cluster and all pods running on it.
