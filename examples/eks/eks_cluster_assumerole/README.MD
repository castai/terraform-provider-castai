## EKS and CAST AI example with CAST AI managed IAM policies

Following example shows how to onboard EKS cluster to CAST AI with CAST AI managed IAM policies.

IAM policies in the example are created by [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-iam).

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create EKS cluster - `eks.tf`
3. Create IAM and other CAST AI related resources to connect EKS cluster to CAST AI - `castai.tf`

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