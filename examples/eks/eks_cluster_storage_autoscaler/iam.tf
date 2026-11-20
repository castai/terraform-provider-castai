# IAM resources required by the CAST AI Storage Autoscaler and by Kvisor storage metrics.
#
# Both components call AWS APIs from inside the cluster. Access is granted via
# IAM Roles for Service Accounts (IRSA): every role below is assumed by a
# dedicated Kubernetes service account through the cluster's OIDC provider.
#
#   castai-agent/castai-storage-autoscaler - expands volumes, right-sizes IOPS/throughput
#   castai-agent/castai-kvisor-controller  - reads EBS volume metadata for storage metrics
#
# The roles are intentionally separate, so that Kvisor does not inherit the
# write permissions (ec2:ModifyVolume) of the Storage Autoscaler.
#
# See https://docs.cast.ai/docs/storage-installation and
# https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics for details.

locals {
  # Names are shared with castai.tf, so the IRSA trust policies always match the
  # service accounts created by the CAST AI Helm chart.
  castai_namespace                   = "castai-agent"
  storage_autoscaler_service_account = "castai-storage-autoscaler"
  kvisor_controller_service_account  = "castai-kvisor-controller"

  cluster_oidc_issuer = data.aws_eks_cluster.existing_cluster.identity[0].oidc[0].issuer
  cluster_oidc_host   = replace(local.cluster_oidc_issuer, "https://", "")

  # IAM role names are limited to 64 characters.
  storage_autoscaler_role_name = substr("castai-storage-autoscaler-${var.cluster_name}", 0, 64)
  kvisor_role_name             = substr("castai-kvisor-${var.cluster_name}", 0, 64)
}

# The cluster's OIDC provider must be registered in IAM. EKS registers it
# automatically for newly created clusters; for older clusters follow
# https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html
data "aws_iam_openid_connect_provider" "cluster_oidc" {
  url = local.cluster_oidc_issuer
}

################################################################################
# Storage Autoscaler
################################################################################

data "aws_iam_policy_document" "castai_storage_autoscaler_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.cluster_oidc.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.cluster_oidc_host}:sub"
      values   = ["system:serviceaccount:${local.castai_namespace}:${local.storage_autoscaler_service_account}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.cluster_oidc_host}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "castai_storage_autoscaler" {
  name               = local.storage_autoscaler_role_name
  assume_role_policy = data.aws_iam_policy_document.castai_storage_autoscaler_assume_role.json
}

data "aws_iam_policy_document" "castai_storage_autoscaler" {
  statement {
    sid    = "EBS"
    effect = "Allow"

    actions = [
      # Read current EBS volume state.
      "ec2:DescribeVolumes",
      # Poll in-flight ModifyVolume operations.
      "ec2:DescribeVolumesModifications",
      # Discover per-VM EBS performance caps (performance rightsizing only).
      "ec2:DescribeInstanceTypes",
    ]
    resources = ["*"]
  }

  # Issue size/IOPS/throughput changes only on volumes identified by EKS's own tag.
  # The two Allow statements below implement an OR between the two tag formats.
  statement {
    sid       = "ModifyVolumeForEksClusterNameTag"
    effect    = "Allow"
    actions   = ["ec2:ModifyVolume"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "aws:ResourceTag/eks:eks-cluster-name"
      values   = [var.cluster_name]
    }
  }

  # Alternatively, allow modifying volumes labeled as owned by this Kubernetes cluster.
  statement {
    sid       = "ModifyVolumeForKubernetesClusterTag"
    effect    = "Allow"
    actions   = ["ec2:ModifyVolume"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}"
      values   = ["owned"]
    }
  }

  statement {
    sid       = "CloudWatch"
    effect    = "Allow"
    actions   = ["cloudwatch:GetMetricData"] # Fetch actual IOPS/throughput consumption (performance rightsizing only).
    resources = ["*"]
  }
}

resource "aws_iam_policy" "castai_storage_autoscaler" {
  name        = "castai-storage-autoscaler-${var.cluster_name}"
  description = "Permissions required by CAST AI Storage Autoscaler"
  policy      = data.aws_iam_policy_document.castai_storage_autoscaler.json
}

resource "aws_iam_role_policy_attachment" "castai_storage_autoscaler" {
  role       = aws_iam_role.castai_storage_autoscaler.name
  policy_arn = aws_iam_policy.castai_storage_autoscaler.arn
}

################################################################################
# Kvisor storage metrics
################################################################################

# Kvisor controller collects cloud volume metadata (disk type, size, provisioned
# IOPS) from the EC2 API. Without IRSA it would depend on the node instance role
# being reachable through IMDS, which is often restricted.
data "aws_iam_policy_document" "castai_kvisor_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.cluster_oidc.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.cluster_oidc_host}:sub"
      values   = ["system:serviceaccount:${local.castai_namespace}:${local.kvisor_controller_service_account}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.cluster_oidc_host}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "castai_kvisor" {
  name               = local.kvisor_role_name
  assume_role_policy = data.aws_iam_policy_document.castai_kvisor_assume_role.json
}

data "aws_iam_policy_document" "castai_kvisor" {
  statement {
    sid       = "EBSVolumes"
    effect    = "Allow"
    actions   = ["ec2:DescribeVolumes"] # Read EBS volume metadata (type, size, IOPS, throughput).
    resources = ["*"]
  }
}

resource "aws_iam_policy" "castai_kvisor" {
  name        = "castai-kvisor-${var.cluster_name}"
  description = "Permissions required by Kvisor to collect EBS volume metrics"
  policy      = data.aws_iam_policy_document.castai_kvisor.json
}

resource "aws_iam_role_policy_attachment" "castai_kvisor" {
  role       = aws_iam_role.castai_kvisor.name
  policy_arn = aws_iam_policy.castai_kvisor.arn
}
