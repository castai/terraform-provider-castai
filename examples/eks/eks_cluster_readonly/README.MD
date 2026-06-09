# Example of EKS cluster connected to CAST AI in READ-ONLY mode (Phase 1)
Following this example creates EKS cluster and its supporting resources using AWS community modules.
After EKS cluster is created it is onboarded to CAST AI in Read-only mode(Phase 1).
Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create CAST AI related resources to connect EKS cluster to CAST AI in read-only mode - `castai.tf`

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region and CAST AI API token

| Variable | Description |
| --- | --- |
| cluster_name                = "" | Name of cluster |
| cluster_region              = "" | Name of region of cluster |
| castai_api_token            = "" | Cast api token |

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
