# IAM user required for CAST.AI

provider "castai" {
  api_token = var.castai_api_token
}

provider "aws" {
  region  = var.region
}

locals {
  iam_user = "castai-eks-${var.cluster_name}"
  aws_account_id = data.aws_caller_identity.current.account_id
  instance_profile = "cast-${substr(var.cluster_name,0,40)}-eks-${substr(var.cluster_id,0,8)}"
}

# RETRIEVE CLUSTER PARAMS AS EKS DATA_SOURCE
data "castai_eks_settings" "eks" {
  account_id = local.aws_account_id
  vpc        = var.vpc_id
  region     = var.region
  cluster    = var.cluster_name
}

data "aws_caller_identity" "current" {}

resource "aws_iam_user" "castai" {
  name = local.iam_user
}

resource "aws_iam_role" "instance_profile_role" {
  name = local.instance_profile

  assume_role_policy = jsonencode({
    Version:"2012-10-17"
    Statement:[
      {
        Sid = ""
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action= ["sts:AssumeRole"]
      }
    ]
  })
}

resource "aws_iam_instance_profile" "instance_profile" {
  name = local.instance_profile
  role = aws_iam_role.instance_profile_role.name
}

resource "aws_iam_role_policy_attachment" "castai_instance_profile_policy" {
  for_each = toset(data.castai_eks_settings.eks.instance_profile_policies)

  role       = aws_iam_instance_profile.instance_profile.role
  policy_arn = each.value
}

resource "aws_iam_access_key" "castai" {
  user = aws_iam_user.castai.name
}

resource "aws_iam_policy" "castai_iam_policy" {
  name   = "CastEKSPolicy-tf"
  policy = data.castai_eks_settings.eks.iam_policy_json
}

resource "aws_iam_user_policy_attachment" "castai_iam_policy_attachment" {
  user       = aws_iam_user.castai.name
  policy_arn = aws_iam_policy.castai_iam_policy.arn
}

resource "aws_iam_user_policy" "castai_user_iam_policy" {
  name   = "castai-user-policy-${var.cluster_name}"
  user   = aws_iam_user.castai.name
  policy = data.castai_eks_settings.eks.iam_user_policy_json
}

resource "aws_iam_user_policy_attachment" "castai_iam_lambda_policy_attachment" {
  for_each = toset(data.castai_eks_settings.eks.lambda_policies)

  user       = aws_iam_user.castai.name
  policy_arn = each.value
}

resource "aws_iam_user_policy_attachment" "castai_user_iam_policy_attachment" {
  for_each = toset(data.castai_eks_settings.eks.iam_managed_policies)

  user       = aws_iam_user.castai.name
  policy_arn = each.key
  depends_on = [aws_iam_policy.castai_iam_policy]
}

output "secret" {
  value = aws_iam_access_key.castai.secret
  sensitive = true
}

