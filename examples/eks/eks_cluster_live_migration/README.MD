## EKS and CAST AI example with CAST AI Autoscaler policies and additional Node Configurations

Following this example shows how to onboard EKS cluster to CAST AI (Phase 2), and configure CAST AI Live Migration capability.

IAM policies required to connect the cluster to CAST AI in the example are created by [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

CAST AI modules are places in one umbrella module that can be reused for different clusters and enabled/disabled

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create IAM and other CAST AI related resources to connect EKS cluster to CAST AI, configure Autoscaler and Node Configurations  - `castai.tf` and `modules/castai`.

CLM is enabled by having:
1. `clm_enabled = true` in `modules/castai/main.tf` in the CAST AI node template.
2. `container_runtime = "containerd"` and `eks_image_family  = "al2023"` in the CAST AI node configuration.
3. `install_live = true` (`install_live = var.install_helm_live`, default value is `true`) in the CAST AI module configuration (`module "castai-eks-cluster"`).
All are in the `modules/castai/main.tf` file.

All other EKS/CAST AI resources mostly manage the EKS cluster and its connection to CAST AI, not specifically the CLM feature.

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region and CAST AI API token.

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

> **Note**
>
> If you are onboarding existing cluster to CAST AI you need to also update [aws-auth](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) configmap. In the configmap instance profile
> used by CAST AI has to be present. Example of entry can be found [here](https://github.com/castai/terraform-provider-castai/blob/157babd57b0977f499eb162e9bee27bee51d292a/examples/eks/eks_cluster_autoscaler_polices/eks.tf#L28-L38).
