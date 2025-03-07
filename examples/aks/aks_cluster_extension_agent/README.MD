# Example of AKS cluster connected to Cast AI via Azure k8s extension
Following example shows how to onboard AKS cluster to CAST AI, via the use of free plan azure extension which deploys Cast AI agent.


Example configuration should be analysed in the following order:
1. Create Virtual network - `vnet.tf`
2. Create AKS cluster - `aks.tf`
3. Accept free plan terms and create azure extension - `extension.tf`
4. Cast AI needed cloud permissions - `iam.tf`
5. Create CAST AI related resources to connect AKS cluster to CAST AI and remaining Castware components - `castai.tf`

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster region, resource_group, subscription ID, tenant ID and CAST AI API token.
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
