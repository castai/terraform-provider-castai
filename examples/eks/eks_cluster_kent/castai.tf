data "aws_caller_identity" "current" {}

resource "castai_eks_clusterid" "this" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = module.eks.cluster_name
}

resource "castai_eks_user_arn" "this" {
  cluster_id = castai_eks_clusterid.this.id
}

module "castai_eks_role_iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.this.arn

  create_iam_resources_per_cluster = true
}

# EKS v21 module defaults to authentication_mode = API_AND_CONFIG_MAP, so the
# CAST AI instance-profile-role principal needs an EKS access entry to be able
# to join nodes to the cluster.
resource "aws_eks_access_entry" "castai" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai_eks_role_iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Pod Identity associations for in-cluster CAST AI workloads that need AWS API
# access (ec2:DescribeInstances, eks:DescribeCluster). Without these,
# castai-agent's AWS SDK falls back to the MNG node instance profile, which
# lacks ec2:DescribeInstances and fails during cluster registration.
#
# We can't reuse the castai_eks_role_iam instance profile role here — its trust
# policy only allows ec2.amazonaws.com (it's an EC2 instance role for nodes that
# CAST AI provisions). Pod Identity requires pods.eks.amazonaws.com as the trust
# principal, so we create a dedicated role for in-cluster workloads.
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

resource "aws_iam_role_policy_attachment" "castai_workload_ec2_read" {
  role       = aws_iam_role.castai_workload.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess"
}

resource "aws_iam_role_policy" "castai_workload_eks_describe" {
  name = "EKSDescribeCluster"
  role = aws_iam_role.castai_workload.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["eks:DescribeCluster"]
      Resource = "*"
    }]
  })
}

locals {
  castai_pod_identity_service_accounts = [
    "castai-agent",
    "castai-cluster-controller",
    "castai-spot-handler",
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
    module.castai_eks_role_iam,
    aws_eks_access_entry.castai,
    aws_eks_pod_identity_association.castai,
    # Forces destroy order: castai pods drain before the karpenter NodeClaim
    # drain runs. Otherwise, draining NodeClaims first would evict castai pods
    # mid-uninstall.
    null_resource.karpenter_drain,
  ]
}

