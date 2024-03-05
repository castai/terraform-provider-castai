# Example of GKE cluster connected to CAST AI in READ-ONLY mode
Following example creates GKE cluster and its supporting resources using GKE community modules.
After GKE cluster is created it is onboarded to CAST AI in Read-only mode.
Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create GKE cluster - `gke.tf`
3. Create CAST AI related resources to connect GKE cluster to CAST AI in read-only mode - `castai.tf`

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
