## AKS and CAST AI example with CAST AI Live Migration (CLM)

Following this example shows how to onboard an AKS cluster to CAST AI and configure CAST AI Live Migration capability.

CAST AI modules are placed in one umbrella module that can be reused for different clusters and enabled/disabled.

Example configuration should be analysed in the following order:
1. Create Virtual network and resource group - `main.tf`
2. Create AKS cluster - `main.tf`
3. Create CAST AI related resources to connect AKS cluster to CAST AI, configure Autoscaler, Node Configurations, and Live Migration - `main.tf` and `module/castai/castai.tf`.

CLM is enabled by having:
1. `clm_enabled = true` in `module/castai/castai.tf` in the CAST AI node template.
2. `install_live = true` (`install_live = var.install_helm_live`, default value is `true`) in the CAST AI module configuration (`module "castai-aks-cluster"`).
Both are in the `module/castai/castai.tf` file.

All other AKS/CAST AI resources mostly manage the AKS cluster and its connection to CAST AI, not specifically the CLM feature.

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region, subscription ID, and CAST AI API token.

| Variable | Description |
| --- | --- |
| cluster_name       | Name of the AKS cluster |
| cluster_region     | Azure region for the cluster |
| subscription_id    | Azure subscription ID |
| castai_api_token   | CAST AI API token |

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
