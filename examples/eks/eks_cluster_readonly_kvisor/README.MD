# Example of existing EKS cluster connected to CAST AI in READ-ONLY mode with Kvisor storage metrics

Following this example onboards an existing EKS cluster to CAST AI in read-only mode (Phase 1) and installs the CAST AI Kvisor component to collect storage metrics.

> **Prerequisites**
> - An existing EKS cluster.
> - IAM Roles for Service Accounts (IRSA) enabled on the cluster, i.e. an OIDC identity provider associated with the EKS cluster.
> - AWS CLI configured with credentials that can access the cluster.

Kvisor uses IAM Roles for Service Accounts (IRSA) to assume an AWS IAM role that is allowed to call `ec2:DescribeVolumes`. The role is trusted by the `castai-kvisor-controller` service account in the `castai-agent` namespace via the EKS OIDC provider.

Example configuration should be analysed in the following order:
1. Configure providers and data sources - `providers.tf`
2. Configure CAST AI related resources to connect the existing EKS cluster to CAST AI in read-only mode and install Kvisor - `castai.tf`

## Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region, CAST AI API token, and optional Kvisor gRPC address

| Variable | Description | Default |
| --- | --- | --- |
| cluster_name | Name of the existing EKS cluster | - |
| cluster_region | AWS region of the existing EKS cluster | - |
| castai_api_token | CAST AI API token created in console.cast.ai API Access keys section | - |
| castai_api_url | CAST AI API URL | https://api.cast.ai |
| kvisor_grpc_addr | CAST AI Kvisor gRPC address | kvisor.prod-master.cast.ai:443 |

3. Initialize Terraform. Under example root folder run:
```
terraform init
```
4. Run Terraform apply:
```
terraform apply
```
5. To destroy resources created by this example:
```
terraform destroy
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
