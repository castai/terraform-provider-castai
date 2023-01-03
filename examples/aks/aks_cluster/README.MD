# Example of AKS cluster connected to CAST AI
Following example creates AKS cluster and its supporting resources.
After AKS cluster is created it is onboarded to CAST AI without enabling [Autoscaler policies](https://docs.cast.ai/product-overview/console/autoscaler/#overview-of-autoscaler-policies) and no custom [node configurations](https://docs.cast.ai/guides/node-configuration/). 
Example configuration should be analysed in the following order:
1. Create Virtual network - `vnet.tf`
2. Create AKS cluster - `aks.tf`
3. Create CAST AI related resources to connect AKS cluster to CAST AI - `castai.tf`

# Usage
1. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token.
2. Initialize Terraform. Under example root folder run:
```
terraform init
```
3. Run Terraform apply:
```
terraform apply -var-file=tf.vars 
```
4. To destroy resources created by this example:
```
terraform destroy -var-file=tf.vars 
```
