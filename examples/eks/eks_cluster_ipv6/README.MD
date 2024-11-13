## EKS and CAST AI example for onboarding IPv6 enabled cluster

Following this example shows how to onboard an IPv6 enabled EKS cluster to CAST AI platform.

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token, etc.

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
