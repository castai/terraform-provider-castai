# Existing AKS cluster and CAST AI example using Azure Workload Identity impersonation

This example onboards an existing AKS cluster to CAST AI using [Azure Workload Identity impersonation](https://docs.cast.ai/docs/sa-impersonation-aks) instead of an App Registration with a client secret.

For prerequisites, how workload identity impersonation works, migrating an existing cluster, and troubleshooting, refer to the [documentation](https://docs.cast.ai/docs/sa-impersonation-aks).

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, resource_group, cluster region, subnets and CAST AI API token.
3. Initialize and apply:
```
terraform init
terraform apply
```
4. To destroy resources created by this example:
```
terraform destroy
```
