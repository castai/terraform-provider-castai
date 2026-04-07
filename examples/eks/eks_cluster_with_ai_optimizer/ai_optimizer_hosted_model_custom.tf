locals {
  model_registry_iam_user_name = "castai-${substr(sha256("${var.castai_organization_id}|${var.custom_model_registry_bucket}"), 0, 12)}"
}

resource "aws_iam_user" "model_registry" {
  count = var.deploy_custom_model ? 1 : 0

  name = local.model_registry_iam_user_name

  tags = {
    Name    = "CAST AI S3 ReadOnly - ${var.custom_model_registry_bucket}"
    Purpose = "CAST AI read-only access to S3 bucket ${var.custom_model_registry_bucket}"
  }
}

resource "aws_iam_user_policy" "model_registry" {
  count = var.deploy_custom_model ? 1 : 0

  name = "CastAIReadOnlyPolicy"
  user = aws_iam_user.model_registry[0].name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:GetObjectAttributes",
          "s3:GetObjectVersionAttributes",
          "s3:HeadObject",
          "s3:ListBucket",
          "s3:ListBucketVersions",
          "s3:GetBucketLocation",
        ]
        Resource = [
          "arn:aws:s3:::${var.custom_model_registry_bucket}",
          "arn:aws:s3:::${var.custom_model_registry_bucket}/*",
        ]
      },
    ]
  })
}

resource "aws_iam_access_key" "model_registry" {
  count = var.deploy_custom_model ? 1 : 0

  user = aws_iam_user.model_registry[0].name
}

resource "castai_ai_optimizer_model_registry" "custom_models" {
  count = var.deploy_custom_model ? 1 : 0

  bucket  = var.custom_model_registry_bucket
  region  = var.custom_model_registry_region
  prefix  = "models"
  credentials = jsonencode({
    access_key_id     = aws_iam_access_key.model_registry[0].id
    secret_access_key = aws_iam_access_key.model_registry[0].secret
  })

  depends_on = [aws_iam_user_policy.model_registry]
}

resource "castai_ai_optimizer_model_specs" "custom_model" {
  count = var.deploy_custom_model ? 1 : 0

  model         = var.custom_model_name
  registry_type = "PRIVATE"

  private_registry {
    base_model_id = var.custom_base_model_spec_id
    registry_id   = castai_ai_optimizer_model_registry.custom_models[0].id
  }
}

resource "castai_ai_optimizer_hosted_model" "custom_model" {
  count = var.deploy_custom_model ? 1 : 0

  cluster_id     = castai_eks_clusterid.cluster_id.id
  model_specs_id = castai_ai_optimizer_model_specs.custom_model[0].id
  service        = "${var.custom_model_name}-service"
  port           = 8080

  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 2
    target_metric = "GPU_CACHE_USAGE_PERCENTAGE"
    target_value  = 5
  }

  depends_on = [module.castai-eks-cluster]
}
