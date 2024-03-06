## GKE and CAST AI example with CAST AI Autoscaler evictor advanced config

Following example shows how to onboard GKE cluster to CAST AI, configure [Autoscaler evictor advanced config](https://docs.cast.ai/docs/evictor-advanced-configuration) 

IAM policies required to connect the cluster to CAST AI in the example are created by [castai/gke-role-iam/castai module](https://github.com/castai/terraform-castai-gke-iam).

This example builds on top of gke_cluster_autoscaler_policies example. Please refer to it for more details.

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create GKE cluster - `gke.tf`
3. Create IAM and other CAST AI related resources to connect GKE cluster to CAST AI, configure Autoscaler and Node Configurations  - `castai.tf`

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your project name, cluster name, cluster region and CAST AI API token.
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


Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
