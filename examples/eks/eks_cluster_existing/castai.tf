# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.cluster_name # Replace with the actual name of your EKS cluster
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}


provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.existing_cluster.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.existing_cluster.certificate_authority.0.data)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      # This requires the awscli to be installed locally where Terraform is executed.
      args = ["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region, "--profile", var.profile]
    }
  }
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

  node_configurations = {
    default = {
      subnets = var.subnets
      tags    = var.tags
      security_groups = [
        var.cluster_security_group_id,
        var.node_security_group_id
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }
  }


  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        spot_interruption_predictions_enabled = false
        spot_interruption_predictions_type    = "aws-rebalance-recommendations"
      }
    }
  }

  autoscaler_settings = {
    enabled                                 = false
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = false
    }

    node_downscaler = {
      enabled = false

      empty_nodes = {
        enabled = false
      }

      evictor = {
        aggressive_mode           = false
        cycle_interval            = "5m10s"
        dry_run                   = false
        enabled                   = false
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = false

      cpu = {
        max_cores = 20
        min_cores = 1
      }

      spot_backups = {
        enabled                          = false
        spot_backup_restore_rate_seconds = 1800
      }
    }
  }

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam]
}

resource "castai_rebalancing_schedule" "spots" {
  name = "rebalance spots at every 30th minute"
  schedule {
    cron = "*/30 * * * *"
  }
  trigger_conditions {
    savings_percentage = 20
  }
  launch_configuration {
    # only consider instances older than 5 minutes
    node_ttl_seconds         = 300
    num_targeted_nodes       = 3
    rebalancing_min_nodes    = 2
    keep_drain_timeout_nodes = false
    selector = jsonencode({
      nodeSelectorTerms = [{
        matchExpressions = [
          {
            key      = "scheduling.cast.ai/spot"
            operator = "Exists"
          }
        ]
      }]
    })
    execution_conditions {
      enabled                     = true
      achieved_savings_percentage = 10
    }
  }
}

resource "castai_rebalancing_job" "spots" {
  cluster_id              = castai_eks_clusterid.cluster_id.id
  rebalancing_schedule_id = castai_rebalancing_schedule.spots.id
  enabled                 = true
  depends_on              = [module.castai-eks-cluster]
}
