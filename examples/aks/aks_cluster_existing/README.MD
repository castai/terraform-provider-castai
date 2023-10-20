# Existing AKS cluster and CAST AI example with CAST AI Autoscaler policies and additional Node Configurations
Following example shows how to onboard existing AKS cluster to CAST AI, configure [Autoscaler policies](https://docs.cast.ai/reference/policiesapi_upsertclusterpolicies) and additional [Node Configurations](https://docs.cast.ai/docs/node-configuration/).

Example configuration should be analysed in the following order:
1. Create CAST AI related resources to connect AKS cluster to CAST AI, configure Autoscaler and Node Configurations - `castai.tf`

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster_rg, cluster region, subnets and CAST AI API token.
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
