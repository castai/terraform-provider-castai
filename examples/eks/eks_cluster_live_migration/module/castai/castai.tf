locals {
  role_name = "castai-eks-role"

  default_node_cfg = {
    default = {
      subnets              = var.subnets
      tags                 = var.tags
      security_groups      = var.security_groups
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }
  }

  default_node_tmpl = {
    default_by_castai = {
      name = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations[
        "default"
      ]
      is_default   = true
      is_enabled   = true
      should_taint = false

      constraints = {
        on_demand          = true
        spot               = true
        use_spot_fallbacks = true

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        spot_interruption_predictions_enabled = true
        spot_interruption_predictions_type    = "aws-rebalance-recommendations"
      }
    }
  }

  node_configuration = merge(local.default_node_cfg, {
    live = {
      subnets              = var.subnets,
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      security_groups      = var.security_groups
      init_script = base64encode(templatefile("${path.module}/eks-init-script.sh", {
        live_proxy_version = trimspace(var.live_proxy_version)
      }))
      container_runtime = "containerd"
      eks_image_family  = "al2023"
    }
  })

  node_templates = merge(local.default_node_tmpl, {
    live_tmpl = {
      configuration_id = module.castai-eks-cluster.castai_node_configurations["live"]
      is_enabled       = true
      should_taint     = true
    }
  })
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = var.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

module "castai-eks-cluster" {
  source = "castai/eks-cluster/castai"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = local.node_configuration

  node_templates = local.node_templates

  autoscaler_settings = {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true
    }

    node_downscaler = {
      enabled = true

      empty_nodes = {
        enabled = true
      }

      evictor = {
        aggressive_mode           = false
        cycle_interval            = "5m10s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = true

      cpu = {
        max_cores = 20
        min_cores = 1
      }
    }
  }

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam]
}

resource "helm_release" "live-helm" {
  name  = "castai-live"
  count = var.install_helm_live ? 1 : 0

  repository = "https://castai.github.io/helm-charts"
  chart      = "castai-live"
  version    = var.live_helm_version

  namespace         = "castai-live"
  create_namespace  = true
  dependency_update = true

  set {
    name  = "castai-aws-vpc-cni.enabled"
    value = "true"
  }

  set {
    name  = "castai.clusterID"
    value = castai_eks_clusterid.cluster_id.id
  }

  set {
    name  = "castai.apiKey"
    value = var.castai_api_token
  }

  set {
    name  = "castai.apiURL"
    value = var.castai_api_url
  }

  wait = false

  depends_on = [module.castai-eks-cluster]
}
