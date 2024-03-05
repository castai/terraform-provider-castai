## Example of creating GCP IAM resources

Following example shows how to create GCP IAM resources required to connect GKE with CAST AI using [castai_gke_iam](https://registry.terraform.io/modules/castai/gke-iam/castai/latest) module.

When creating a service account you can enforce conditional, attribute-based access on `iam.serviceAccountUser` role.
It can access and act as all other service accounts or be scoped to the ones used by node pools in the GKE cluster, which is more secure and therefore recommended.

### Steps to take to successfully create GCP IAM resources with `iam.serviceAccountUser` role and custom condition.

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Rename `tf_scoped.vars.example` to `tf.vars`
2. To get `service_accounts_unique_ids` run :
```
PROJECT_ID=<PLACEHOLDER> LOCATION=<PLACEHOLDER> CLUSTER_NAME=<PLACEHOLDER> ./script.sh
```
3. Update `tf.vars` file with your project name, cluster name, cluster region, service accounts unique ids and CAST AI API token.
4. Initialize Terraform. Under example root folder run:
```
terraform init
```
5. Run Terraform apply:
```
terraform apply -var-file=tf.vars
```
6. To destroy resources created by this example:
```
terraform destroy -var-file=tf.vars
```

### Steps to take to successfully create GCP IAM resources with default `iam.serviceAccountUser` without custom condition.

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Rename `tf_default.vars.example` to `tf.vars`
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
