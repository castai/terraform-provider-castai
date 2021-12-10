# IAM user required for CAST.AI

provider "castai" {
  api_url   = "https://api.dev-master.cast.ai"
  api_token = var.castai_api_token
}

locals {
  cluster_vpc    = module.vpc.vpc_id
  aws_account_id = data.aws_caller_identity.current.account_id
}

# provides up-to-date permissions for new iam user (user policy json, iam policy json and managed services required).
data "castai_credentials_eks" "eks" {
  account_id = local.aws_account_id
  vpc        = local.cluster_vpc
  region     = var.cluster_region
  cluster    = var.cluster_name
}

data "aws_caller_identity" "current" {}

resource "aws_iam_user" "castai" {
  name = "castai-eks-${var.cluster_name}"
}

resource "aws_iam_access_key" "castai" {
  user = aws_iam_user.castai.name

  depends_on = [aws_iam_user.castai]
}

resource "aws_iam_policy" "castai_iam_policy" {
  name   = "CastaiEKSPolicy-tf"
  policy = data.castai_credentials_eks.iam_policy_json
}

resource "aws_iam_user_policy" "castai_user_iam_policy" {
  name   = "castai-user-policy-${var.cluster_name}"
  user   = aws_iam_user.castai.name
  policy = data.castai_credentials_eks.iam_user_policy_json
}

resource "aws_iam_user_policy_attachment" "castai_iam_policy_attachment" {
  user       = aws_iam_user.castai.name
  policy_arn = aws_iam_policy.castai_iam_policy.arn
}

resource "aws_iam_user_policy_attachment" "castai_user_iam_policy_attachment" {
  for_each = toset(data.castai_credentials_eks.iam_managed_policies)

  user       = aws_iam_user.castai.name
  policy_arn = each.key
  depends_on = [aws_iam_policy.castai_iam_policy]
}

output "secret" {
  value = aws_iam_access_key.castai.encrypted_secret
}

