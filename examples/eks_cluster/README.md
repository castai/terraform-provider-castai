## EKS and CAST AI setup example

Reference setup for new or existing EKS cluster connected with CAST.AI

### Instruction for full setup (create vpc, eks and connect with cast.ai)

- `terraform apply -target module.vpc` (needs to be created before creating further iam and CAST.AI resources)
- `terraform apply` (applying the rest of resources)

### Connecting existing EKS cluster to CAST AI

- Take setup of aws iam credential for CAST.AI from [aws_iam_castai.tf](castai_aws_iam.tf)
- Take setup of CAST.AI cluster and agent installation from [main.tf](main.tf)
- Specify created aws iam users and roles in your setup to access your cluster [see your_eks.tf](your_eks.tf)

```terraform
module "eks" {
  source = "terraform-aws-modules/eks/aws"

  // ...

  map_users = [
    // ...
    {
      userarn  = aws_iam_user.castai.arn
      username = aws_iam_user.castai.name
      groups   = ["system:masters"]
    },
    // ...
  ]

  map_roles = [
    // ...
    {
      rolearn  = aws_iam_role.instance_profile_role.arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups   = ["system:bootstrappers", "system:nodes"]
    },
    // ...
  ]

  // ...
  
}
```

### Troubleshooting

##### Policy already exists

If you have already onboarded another cluster in the same account, you will have created `CastEKSPolicy-tf` before. By
trying to create the policy again, you might receive error:

```
Error: error creating IAM policy CastEKSPolicy-tf: EntityAlreadyExists: A policy called CastEKSPolicy-tf already exists. Duplicate names are not allowed.
```

Either reuse the existing one (point dependant resources to it) or import it into your terraform state using aws
provider this way:

```shell
$ terraform import aws_iam_policy.castai_iam_policy arn:aws:iam::{{YOUR_ACCOUNT_ID}}:policy/CastEKSPolicy-tf
```

##### Role with name castai-eks-tf-eks1 already exists

Same as with policies, either reuse your existing cast-eks-{specify-cluster_name} role or import it:

```shell
terraform import aws_iam_role.instance_profile_role castai-eks-{specify-cluster_name}
```

##### User with name castai-eks-{specify-cluster_name} already exists.

```shell
$ terraform import aws_iam_user.castai castai-eks-{specify-cluster_name}
```

More info regarding aws iam resources can be found in official terraform/aws docs:
https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources
