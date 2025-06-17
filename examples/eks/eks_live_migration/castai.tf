data "aws_caller_identity" "current" {}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

resource "aws_eks_access_entry" "access_entry" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.region
  cluster_name = var.cluster_name
  depends_on   = [module.eks, helm_release.calico, aws_eks_access_entry.access_entry]
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

module "castai-eks-cluster" {
  source = "castai/eks-cluster/castai"

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  aws_account_id         = data.aws_caller_identity.current.account_id
  aws_cluster_region     = var.region
  aws_cluster_name       = module.eks.cluster_name
  aws_assume_role_arn    = module.castai-eks-role-iam.role_arn
  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  // Default node configuration will be used for all CAST provisioned nodes unless specific configuration is requested.
  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      subnets              = module.vpc.private_subnets
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      security_groups = [
        module.eks.node_security_group_id,
      ]
      init_script       = base64encode(file("eks-init-script.sh"))
      container_runtime = "containerd"
      eks_image_family  = "al2023"
    }
  }

  node_templates = {
    # Already contains live binaries on nodes
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand                     = true
        spot                          = true
        use_spot_fallbacks            = true
        fallback_restore_rate_seconds = 1800

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        architectures = ["amd64"]
      }
    }

    # Same setup as default, but with the goal to forcefully bring nodes with Live binaries installed, based on the NT node selector
    live-enabled = {
      name             = "live-enabled"
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand                     = true
        spot                          = true
        use_spot_fallbacks            = true
        fallback_restore_rate_seconds = 1800

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        architectures = ["amd64"]
      }
    }
  }

  autoscaler_settings = {
    enabled                                 = true
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
        cycle_interval            = "5s10s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = true

      cpu = {
        max_cores = 100
        min_cores = 1
      }
    }
  }
}