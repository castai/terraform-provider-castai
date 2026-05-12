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

  values = [yamlencode({
    kent = {
      enabled = true
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
  ]
}

