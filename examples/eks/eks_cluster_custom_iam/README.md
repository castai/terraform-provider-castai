## EKS and CAST AI example using custom IAM policies

Following example shows how to onboard EKS cluster to CAST AI using custom IAM policies that can be tailored for
your needs.

IAM policies in the example allows CAST AI:

- start new instances only in private subnets of cluster VPC
- start new instances only with predefined security groups
- manage existing instances that are part of the cluster
- other necessary permissions scoped to specific cluster
- all permissions are scoped to specific EKS cluster

Most of the permissions are mandatory for CAST AI to work properly.
Optional permissions include:

- `ec2.ImportKeyPair` - needed only if you will provide public SSH key for CAST nodes
- `autoscaling:SuspendProcesses` - needed only if you will use pause/resume functionality
- `autoscaling:ResumeProcesses` - needed only if you will use pause/resume functionality

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create IAM resources required for CAST AI connection - `iam.tf`
4. Create CAST AI related resources to connect EKS cluster to CAST AI - `castai.tf`

# Usage
1. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token
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
