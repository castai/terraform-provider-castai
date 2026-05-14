locals {
  karpenter_namespace     = "karpenter"
  karpenter_chart_version = "1.7.1"
}

# Karpenter IAM, instance profile, SQS interruption queue, EventBridge rules.
module "karpenter" {
  source  = "terraform-aws-modules/eks/aws//modules/karpenter"
  version = "~> 21.0"

  cluster_name = module.eks.cluster_name
  namespace    = local.karpenter_namespace

  # The v1 controller policy exceeds the 6144-byte standard-IAM-policy quota.
  # Inline policies allow up to 10240 bytes.
  enable_inline_policy = true

  # node_iam_role_name must match the role name referenced by EC2NodeClass.spec.role
  node_iam_role_use_name_prefix   = false
  node_iam_role_name              = local.name
  create_pod_identity_association = true

  tags = local.tags
}

# Karpenter requires this for some discovery flows.
resource "aws_iam_role_policy" "karpenter_list_instance_profiles" {
  name = "KarpenterListInstanceProfiles"
  role = module.karpenter.iam_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid      = "AllowListInstanceProfiles"
      Effect   = "Allow"
      Action   = "iam:ListInstanceProfiles"
      Resource = "*"
    }]
  })
}

resource "helm_release" "karpenter" {
  name                = "karpenter"
  namespace           = local.karpenter_namespace
  create_namespace    = true
  repository          = "oci://public.ecr.aws/karpenter"
  repository_username = data.aws_ecrpublic_authorization_token.token.user_name
  repository_password = data.aws_ecrpublic_authorization_token.token.password
  chart               = "karpenter"
  version             = local.karpenter_chart_version

  # CRDs are part of the chart manifests and are POSTed to the API server
  # before helm returns regardless of `wait`. `wait` only governs *readiness*
  # of Deployments/Pods, which kubectl_manifest doesn't need (the gavinbunney
  # provider retries on transient `no matches for kind` during discovery).
  # Skipping the readiness wait keeps apply snappy and — more importantly —
  # makes destroy order predictable: see the pre-delete-hook comment in
  # castai.tf for why a fast-returning karpenter uninstall matters there.
  wait = false

  values = [yamlencode({
    nodeSelector = {
      "karpenter.sh/controller" = "true"
    }
    tolerations = [
      {
        key      = "CriticalAddonsOnly"
        operator = "Exists"
      },
      {
        key      = "karpenter.sh/controller"
        operator = "Exists"
        effect   = "NoSchedule"
      },
    ]
    settings = {
      clusterName       = module.eks.cluster_name
      clusterEndpoint   = module.eks.cluster_endpoint
      interruptionQueue = module.karpenter.queue_name
    }
    webhook = {
      enabled = false
    }
  })]

  # No `lifecycle { ignore_changes = [repository_password] }`. The data source
  # returns a 12h-TTL token; freezing it in state via ignore_changes means any
  # re-apply >12h after the initial apply fails at OCI login with HTTP 403
  # "Your authorization token has expired."
  #
  # Trade-off: every plan shows a 1-attribute diff on `repository_password` as
  # the data source re-resolves. Applying that diff triggers a helm release
  # revision bump (~7s, no-op upgrade), but the underlying Karpenter pods are
  # not restarted because `version` is pinned and the rendered manifests are
  # identical. Confirmed empirically: pod names and creationTimestamps survive
  # re-apply.
}
