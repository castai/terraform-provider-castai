# Example of EKS cluster connected to CAST AI with enabled Kvisor security agent
Following this example creates EKS cluster and its supporting resources using AWS community modules.\
After EKS cluster is created it is onboarded to CAST AI.\
[Kvisor security agent](https://docs.cast.ai/docs/kvisor) is deployed to the cluster and security policies are enabled.\
See `install_security_agent` and `kvisor_values` variables in `castai.tf` file.\
Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create CAST AI related resources to connect EKS cluster to CAST AI with Kvisor enabled - `castai.tf`

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

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
