# GKE clusters in terraform workspaces using resource from rebalancing schedule from organization workspace

This example onboards existing GKE clusters created in different terraform workspaces to CAST AI,
with usage of rebalancing schedule which is created in another terraform workspace.

## Usage

1. Rename:
   - `tf.vars.example` to `tf.vars`
   - `tf_clusterA.vars.example` to `tf_clusterA.vars`
   - `tf_clusterB.vars.example` to `tf_clusterB.vars`

   e.g.

   ```
   cp tf.vars.example tf.vars
   cp tf_clusterA.vars.example tf_clusterA.vars
   cp tf_clusterB.vars.example tf_clusterB.vars
   ```

2. Update `tf.vars`, `tf_clusterA.vars`, `tf_clusterB.vars`

**Note:** please see [this][gcp-iam-doc] instruction to configure `service_accounts_unique_ids`

[gcp-iam-doc]: https://github.com/castai/terraform-provider-castai/tree/master/examples/gke/gke_castai_iam#steps-to-take-to-successfully-create-gcp-iam-resources-with-iamserviceaccountuser-role-and-custom-condition

3. Initialize Terraform. Under example root folder run:
```
terraform init
```

4. Create organization workspace and create resource in organization workspace

```
terraform workspace new org-workspace
terraform plan -var-file=tf.vars
terraform apply -var-file=tf.vars
```

5. Create workspace for the first cluster and create resources in this workspace

```
terraform workspace new clusterA
terraform plan -var-file=tf_clusterA.vars
terraform apply -var-file=tf_clusterA.vars
```

6. Create workspace for the second cluster and create resources in this workspace

```
terraform workspace new clusterB
terraform plan -var-file=tf_clusterB.vars
terraform apply -var-file=tf_clusterB.vars
```

7. Open CAST AI console and check that clusters are using the same configuration for Rebalancing Schedule

8. To destroy resources created by this example:
```
terraform workspace select org-workspace
terraform destroy -var-file=tf.vars

terraform workspace select clusterA
terraform destroy -var-file=tf_clusterA.vars

terraform workspace select clusterB
terraform destroy -var-file=tf_clusterB.vars
```
