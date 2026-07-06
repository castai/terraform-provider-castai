# Connect existing EKS cluster to CAST AI in read-only mode and install Kvisor for storage metrics.

data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.cluster_name
}

data "aws_iam_openid_connect_provider" "eks" {
  url = data.aws_eks_cluster.existing_cluster.identity[0].oidc[0].issuer
}

locals {
  oidc_issuer = trimprefix(data.aws_eks_cluster.existing_cluster.identity[0].oidc[0].issuer, "https://")
}

# Configure EKS cluster connection to CAST AI in read-only mode.
resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name
}

resource "helm_release" "castai_agent" {
  name             = "castai-agent"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-agent"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true

  set = concat(
    [
      {
        name  = "provider"
        value = "eks"
      },
      {
        # Required until https://github.com/castai/helm-charts/issues/135 is fixed.
        name  = "createNamespace"
        value = "false"
      },
    ],
    var.castai_api_url != "" ? [{
      name  = "apiURL"
      value = var.castai_api_url
    }] : [],
  )

  set_sensitive = [
    {
      name  = "apiKey"
      value = castai_eks_cluster.this.cluster_token
    },
  ]
}

# IAM role for Kvisor controller (IRSA) to describe EBS volumes for storage metrics.
data "aws_iam_policy_document" "kvisor_controller_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    effect  = "Allow"

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.eks.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer}:sub"
      values   = ["system:serviceaccount:castai-agent:castai-kvisor-controller"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "kvisor_controller" {
  name               = "${var.cluster_name}-castai-kvisor-controller"
  assume_role_policy = data.aws_iam_policy_document.kvisor_controller_assume_role.json
}

data "aws_iam_policy_document" "kvisor_controller" {
  statement {
    actions   = ["ec2:DescribeVolumes"]
    resources = ["*"]
    effect    = "Allow"
  }
}

resource "aws_iam_role_policy" "kvisor_controller" {
  name   = "castai-kvisor-controller"
  role   = aws_iam_role.kvisor_controller.id
  policy = data.aws_iam_policy_document.kvisor_controller.json
}

resource "helm_release" "castai_kvisor" {
  name             = "castai-kvisor"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-kvisor"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true

  set = [
    {
      name  = "castai.clusterID"
      value = castai_eks_cluster.this.id
    },
    {
      name  = "castai.grpcAddr"
      value = var.kvisor_grpc_addr
    },
    {
      name  = "controller.serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
      value = aws_iam_role.kvisor_controller.arn
    },
    {
      name  = "controller.serviceAccount.name"
      value = "castai-kvisor-controller"
    },
    {
      name  = "controller.extraArgs.cloud-provider"
      value = "aws"
    },
    {
      name  = "controller.extraArgs.cloud-provider-aws-region"
      value = var.cluster_region
    },
    {
      name  = "controller.extraArgs.cloud-provider-storage-sync-enabled"
      value = "true"
    },
    {
      name  = "agent.enabled"
      value = "true"
    },
    {
      name  = "agent.extraArgs.storage-stats-enabled"
      value = "true"
    },
  ]

  set_sensitive = [
    {
      name  = "castai.apiKey"
      value = castai_eks_cluster.this.cluster_token
    },
  ]
}
