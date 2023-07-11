# 3. Connect EKS cluster to CAST AI.

locals {
  role_name = "castai-eks-role"
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

data "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}


provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      # This requires the awscli to be installed locally where Terraform is executed.
      args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.cluster_region]
    }
  }
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = data.castai_eks_user_arn.castai_user_arn.arn

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

  api_url = var.castai_api_url

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }

    test_node_config = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      kubelet_config = jsonencode({
        "registryBurst" : 20,
        "registryPullQPS" : 10
      })
      container_runtime = "containerd"
      volume_type       = "gp3"
      volume_iops       = 3100
      volume_throughput = 130
      imds_v1           = true
    }
  }

  node_templates = {
    spot_tmpl = {
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      should_taint     = true

      custom_labels = {
        custom-label-key-1 = "custom-label-value-1"
        custom-label-key-2 = "custom-label-value-2"
      }

      custom_taints = [
        {
          key = "custom-taint-key-1"
          value = "custom-taint-value-1"
        },
        {
          key = "custom-taint-key-2"
          value = "custom-taint-value-2"
        }
      ]

      constraints = {
        fallback_restore_rate_seconds = 1800
        spot                          = true
        use_spot_fallbacks            = true
        min_cpu                       = 4
        max_cpu                       = 100
        instance_families = {
          exclude = ["m5"]
        }
        compute_optimized = false
        storage_optimized = false
      }
    }
  }

  # Configure Autoscaler policies as per API specification https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies.
  # Here:
  #  - unschedulablePods - Unscheduled pods policy
  #  - spotInstances     - Spot instances configuration
  #  - nodeDownscaler    - Node deletion policy
  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["aws"],
            "spotBackups": {
                "enabled": true
            },
            "spotDiversityEnabled": false,
            "spotDiversityPriceIncreaseLimitPercent": 20,
            "spotInterruptionPredictions": {
              "enabled": true,
              "type": "AWSRebalanceRecommendations"
            }
        },
        "nodeDownscaler": {
            "enabled": true,
            "emptyNodes": {
                "enabled": true
            },
            "evictor": {
                "aggressiveMode": false,
                "cycleInterval": "5m10s",
                "dryRun": false,
                "enabled": true,
                "nodeGracePeriodMinutes": 10,
                "scopedMode": false
            }
        },
        "clusterLimits": {
            "cpu": {
                "maxCores": 20,
                "minCores": 1
            },
            "enabled": true
        }
    }
  EOT

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
		node_ttl_seconds = 300
		num_targeted_nodes = 3
		rebalancing_min_nodes = 2
		keep_drain_timeout_nodes = false
		selector = jsonencode({
			nodeSelectorTerms = [{
				matchExpressions = [
					{
						key =  "scheduling.cast.ai/spot"
						operator = "Exists"
					}
				]
			}]
		})
		execution_conditions {
			enabled = true
			achieved_savings_percentage = 10
		}
	}
}

resource "castai_rebalancing_job" "spots" {
	cluster_id = castai_eks_clusterid.cluster_id.id
	rebalancing_schedule_id = castai_rebalancing_schedule.spots.id
	enabled = true
}
