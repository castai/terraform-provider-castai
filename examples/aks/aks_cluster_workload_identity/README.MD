# Existing AKS cluster onboarded to CAST AI with Workload Identity

This example shows how to connect an existing AKS cluster to CAST AI using Azure Workload Identity instead of the default App Registration (client secret) method.

Workload Identity uses federated credentials, which means no client secrets are created or rotated. Permissions are scoped to the cluster's resource groups via a User-Assigned Managed Identity.

> **Note:** This feature requires a feature flag to be enabled before `terraform apply` will succeed. `terraform plan` will complete without it, but `terraform apply` will fail with a 403 error. Contact CAST AI support to have it enabled.

The only difference from the standard [aks_cluster_existing](../aks_cluster_existing) example is `authentication_method = "workload_identity"` in the module block.

## What Terraform creates

- A User-Assigned Managed Identity in the cluster's resource group
- A federated identity credential on that managed identity
- A `CastAKSRole` custom role definition
- Role assignments scoped to both the cluster resource group and the node resource group (`MC_*`)

## Migrating an existing cluster

Clusters already onboarded via App Registration can switch to Workload Identity without disconnecting. Add `authentication_method = "workload_identity"` to the module block and run `terraform apply`.

Terraform will destroy the App Registration resources and create the managed identity and federated credential as replacements. Role assignments are also replaced. CAST AI will be temporarily unable to authenticate during propagation — this typically resolves within a few minutes.

## Example configuration order

1. Connect existing AKS cluster to CAST AI with Workload Identity - `castai.tf`

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` with your cluster name, resource group, region, subnets, and CAST AI API token.
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
