data "aws_caller_identity" "current" {}

# Registers the cluster with CAST AI so `tofu destroy` cleanly de-registers it.
# The agent would also self-register on first connect, but having TF own the
# registration means destroy removes the cluster record from CAST AI rather
# than leaving it orphaned in the console.
#
# We deliberately do NOT create castai_eks_user_arn / module.castai_eks_role_iam
# here — those provision a cross-account IAM role for CAST AI's backend to call
# AWS on the customer's behalf (the legacy provisioning path). With Kent,
# Kentroller delegates node provisioning to Karpenter inside the cluster, so
# no backend-side AWS calls happen and the cross-account role is unused.
resource "castai_eks_clusterid" "this" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = module.eks.cluster_name
}

# Pod Identity associations for in-cluster CAST AI workloads that need AWS API
# access. Without these, castai-agent's AWS SDK falls back to the MNG node
# instance profile, which lacks the needed permissions and fails during
# cluster registration.
data "aws_iam_policy_document" "castai_workload_assume" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["pods.eks.amazonaws.com"]
    }
    actions = [
      "sts:AssumeRole",
      "sts:TagSession",
    ]
  }
}

resource "aws_iam_role" "castai_workload" {
  name               = "${local.name}-castai-workload"
  description        = "Used by in-cluster CAST AI workloads via EKS Pod Identity"
  assume_role_policy = data.aws_iam_policy_document.castai_workload_assume.json
  tags               = local.tags
}

# Minimal permission set. The broad AmazonEC2ReadOnlyAccess managed policy was
# replaced with this inline policy after PR review — only the actions actually
# exercised by castai-agent and castai-cluster-controller in the Kent flow.
#
# - ec2:DescribeInstances: agent enriches node info during cluster registration;
#   without it, registration fails with "describing instance_id=i-...:
#   DescribeInstances ... context canceled".
# - ec2:DescribeAvailabilityZones / DescribeInstanceTypes /
#   DescribeInstanceTypeOfferings / DescribeSpotPriceHistory + pricing:GetProducts:
#   direct-inventory feature (AZ / instance-type / spot-price discovery).
# - eks:DescribeCluster: cluster-controller health checks.
resource "aws_iam_role_policy" "castai_workload" {
  name = "castai-workload"
  role = aws_iam_role.castai_workload.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ec2:DescribeAvailabilityZones",
          "ec2:DescribeInstanceTypes",
          "ec2:DescribeInstanceTypeOfferings",
          "ec2:DescribeSpotPriceHistory",
          "pricing:GetProducts",
        ]
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = ["eks:DescribeCluster"]
        Resource = module.eks.cluster_arn
      },
    ]
  })
}

locals {
  castai_pod_identity_service_accounts = [
    "castai-agent",
    "castai-cluster-controller",
  ]
}

resource "aws_eks_pod_identity_association" "castai" {
  for_each = toset(local.castai_pod_identity_service_accounts)

  cluster_name    = module.eks.cluster_name
  namespace       = "castai-agent"
  service_account = each.value
  role_arn        = aws_iam_role.castai_workload.arn

  depends_on = [module.eks]
}

# CAST AI umbrella chart with kent.enabled=true.
#
# The umbrella bundles castai-agent, castai-cluster-controller, castai-kentroller,
# castai-workload-autoscaler (+ exporter), castai-live (CLM, dormant by default),
# castai-pod-mutator, castai-spot-handler, and castai-kvisor under the kent subchart.
#
# A kent-preflight pre-install Job hard-fails if Karpenter (deployment +
# nodepools.karpenter.sh CRD) isn't present — depends_on below guarantees ordering.
#
# Do NOT use the upstream castai/eks-cluster/castai module alongside this — it
# would double-install castai-agent and castai-cluster-controller.
resource "helm_release" "castai" {
  name             = "castai"
  namespace        = "castai-agent"
  create_namespace = true
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai"
  version          = "0.34.13"

  # Nine subcharts + image pulls + Karpenter-provisioned nodes joining ~5-7 min
  # total on first install; the default 300s helm timeout hits exactly at the
  # boundary and returns context-deadline-exceeded while pods are still rolling.
  timeout = 600

  values = [yamlencode({
    kent = {
      enabled = true
      # Disable kent subchart pre-delete hooks that deadlock `tofu destroy`.
      # Two distinct traps the umbrella ships, both surface only at uninstall:
      #
      # 1. No-tolerations trap — castai-workload-autoscaler.preDeleteHook
      #    strips finalizers from Recommendation CRs at uninstall, but the
      #    Job's pod ships without tolerations or nodeSelector. By the time
      #    helm uninstall fires, helm_release.karpenter has already returned
      #    (wait=false) and module.karpenter has stripped the karpenter node
      #    IAM role, so Karpenter-spawned nodes are NotReady and the MNG nodes
      #    carry karpenter.sh/controller taints — nowhere left to schedule.
      #    Helm uninstall hangs at DeadlineExceeded. Leaked Recommendation CRs
      #    are benign: the namespace is deleted right after.
      #
      # 2. No-egress trap — castai-live.castai-aws-vpc-cni ships a
      #    `pre-delete` Job (patch-daemonset-remove) that pulls
      #    ghcr.io/castai/live/kubectl to unpatch the aws-node DaemonSet.
      #    By uninstall time, module.vpc has already destroyed the NAT
      #    gateway, so the pod can't reach ghcr.io and image pull hangs until
      #    the helm timeout. The aws-node patch is a no-op cleanup against a
      #    cluster that's about to disappear anyway. We disable the whole
      #    castai-aws-vpc-cni subchart here because we're running CLM dormant
      #    (controller.replicaCount=0 upstream) — the CAST-forked CNI
      #    DaemonSet isn't doing anything useful and just costs us a hook.
      "castai-workload-autoscaler" = {
        preDeleteHook = {
          enabled = false
        }
      }
      "castai-live" = {
        "castai-aws-vpc-cni" = {
          enabled = false
        }
      }
    }
    global = {
      castai = {
        apiKey   = var.castai_api_token
        apiURL   = var.castai_api_url
        grpcURL  = var.castai_grpc_url
        provider = "eks"
      }
    }
  })]

  depends_on = [
    helm_release.karpenter,
    kubectl_manifest.karpenter_default_nodepool,
    kubectl_manifest.karpenter_default_nodeclass,
    castai_eks_clusterid.this,
    aws_eks_pod_identity_association.castai,
    # Forces destroy order: castai pods drain before the karpenter NodeClaim
    # drain runs. Otherwise, draining NodeClaims first would evict castai pods
    # mid-uninstall.
    null_resource.karpenter_drain,
  ]
}

