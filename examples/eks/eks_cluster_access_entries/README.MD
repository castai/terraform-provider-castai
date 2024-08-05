## EKS and CAST AI example with Access entries as way of auth (Phase 2)

Following this example shows how to onboard an EKS cluster to CAST AI with CAST AI managed IAM policies.

IAM policies in the example are created by [castai/eks-iam-role/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create IAM and other CAST AI related resources to connect EKS cluster to CAST AI (In Phase 2)- `castai.tf`

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token

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
terraform apply -var-file=tf.vars
```
5. To destroy resources created by this example:
```
terraform destroy -var-file=tf.vars
```

> **Note**
>
> Cluster access mode should be either `EKS API and ConfigMap` or `EKS API` in case of existing cluster.

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
