# Example of AKS cluster connected to CAST AI
Following example creates AKS cluster and its supporting resources.
After AKS cluster is created it is onboarded to CAST AI without enabling [Autoscaler policies](https://docs.cast.ai/reference/policiesapi_upsertclusterpolicies) and no custom [node configurations](https://docs.cast.ai/docs/node-configuration/).
Example configuration should be analysed in the following order:
1. Create Virtual network - `vnet.tf`
2. Create AKS cluster - `aks.tf`
3. Create CAST AI related resources to connect AKS cluster to CAST AI - `castai.tf`

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token.
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
