data "aws_caller_identity" "current" {}

resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = module.eks.cluster_name
}

data "aws_iam_policy_document" "castai_kentroller_assume" {
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

resource "aws_iam_role" "castai_kentroller" {
  name               = "${local.name}-castai-kentroller"
  description        = "Used by castai-kentroller via EKS Pod Identity"
  assume_role_policy = data.aws_iam_policy_document.castai_kentroller_assume.json
  tags               = local.tags
}

resource "aws_iam_role_policy" "castai_kentroller" {
  name = "castai-kentroller"
  role = aws_iam_role.castai_kentroller.name
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

resource "aws_eks_pod_identity_association" "castai_kentroller" {
  cluster_name    = module.eks.cluster_name
  namespace       = "castai-agent"
  service_account = "castai-kentroller"
  role_arn        = aws_iam_role.castai_kentroller.arn

  depends_on = [module.eks]
}

resource "helm_release" "castai" {
  name             = "castai"
  namespace        = "castai-agent"
  create_namespace = true
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai"

  timeout = 600

  values = [yamlencode({
    kent = {
      enabled = true
      # Skip the preflight check that expects standalone Karpenter. EKS Auto
      # Mode provides Karpenter CRDs natively, but the preflight Job looks for
      # a Karpenter Deployment which doesn't exist on Auto Mode.
      preflight = {
        enabled = false
      }
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
      # EKS Auto Mode nodes default to IMDS hop limit 1, which blocks pods
      # from reaching instance metadata. The NodeClass CRD has no field to
      # set the hop limit, so provide static values to bypass IMDS entirely.
      "castai-agent" = {
        # additionalEnv = {
        #  EKS_REGION       = var.cluster_region
        #  EKS_CLUSTER_NAME = module.eks.cluster_name
        #  EKS_ACCOUNT_ID   = data.aws_caller_identity.current.account_id
        #}
      }
    }
    global = {
      castai = {
        apiURL   = var.castai_api_url
        grpcURL  = var.castai_grpc_url
        provider = "eks"
      }
    }
  })]

  set_sensitive = [
    {
      name  = "global.castai.apiKey"
      value = castai_eks_cluster.this.cluster_token
    },
  ]

  depends_on = [
    castai_eks_cluster.this,
    aws_eks_pod_identity_association.castai_kentroller,
  ]
}

