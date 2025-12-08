# 3. Connect EKS cluster to CAST AI.

locals {
  role_name = "castai-eks-role"
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}


provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes = {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      # This requires the awscli to be installed locally where Terraform is executed.
      args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.cluster_region]
    }
  }
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

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
  source  = "castai/eks-cluster/castai"
  version = "13.5.1"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration_name = "default"

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
      min_disk_size        = 30
    }

    hyperscale = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      min_disk_size        = 30
    }

    infra = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      min_disk_size        = 30
    }
  }

  node_templates = {
    # Template 1: hyperscale-demand (on-demand, AMD instances)
    hyperscale_demand = {
      name               = "hyperscale-demand"
      configuration_name = "hyperscale"
      is_default         = false
      is_enabled         = true
      should_taint       = false

      custom_labels = {
        "spark-nodeselect-instance-type"  = "amd64-64-16"
        "spark-nodeselect-nodepool-group" = "hyper"
        "spark-nodeselect-preemptible"    = "false"
      }

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        instance_families = {
        }

        architectures = ["amd64"]
        max_cpu       = 8
      }
    }

    # Template 2: default-by-castai (default template, on-demand, multi-arch)
    default_by_castai = {
      name               = "default-by-castai"
      configuration_name = "default"
      is_default         = true
      is_enabled         = true
      should_taint       = false

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        architectures = ["amd64", "arm64"]
        max_cpu       = 8
        instance_families = {
          include = ["c5a", "c5ad", "c6a", "c7a", "m5a", "m5ad", "m6a", "m7a", "m8a", "r5a", "r5ad", "r6a", "r7a", "r8a"]
        }
      }
    }

    # Template 3: infra (infrastructure nodes, on-demand, AMD/INTEL)
    infra = {
      name               = "infra"
      configuration_name = "infra"
      is_default         = false
      is_enabled         = true
      should_taint       = false

      custom_labels = {
        "scheduling.cast.ai/node-template" = "infra"
        "sys-type"                         = "infra"
      }

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        architectures = ["amd64"]
        max_cpu       = 8
        instance_families = {
          include = ["c5a", "c5ad", "c6a", "c7a", "m5a", "m5ad", "m6a", "m7a", "m8a", "r5a", "r5ad", "r6a", "r7a", "r8a"]
        }
      }
    }

    # Template 4: default (on-demand, AMD/INTEL)
    default = {
      name               = "default"
      configuration_name = "default"
      is_default         = false
      is_enabled         = true
      should_taint       = false

      custom_labels = {
        "scheduling.cast.ai/node-template" = "default"
      }

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        architectures = ["amd64"]

        cpu_manufacturers = ["AMD", "INTEL"]
        max_cpu           = 8

        instance_families = {
          include = ["c5a", "c5ad", "c6a", "c7a", "m5a", "m5ad", "m6a", "m7a", "m8a", "r5a", "r5ad", "r6a", "r7a", "r8a"]
        }


      }
    }

    # Template 5: hyperscale-spot (spot instances, AMD)
    hyperscale_spot = {
      name               = "hyperscale-spot"
      configuration_name = "hyperscale"
      is_default         = false
      is_enabled         = true
      should_taint       = false

      custom_labels = {
        "spark-nodeselect-instance-type"  = "amd64-64-16"
        "spark-nodeselect-nodepool-group" = "hyper"
        "spark-nodeselect-preemptible"    = "true"
      }

      constraints = {
        on_demand          = false
        spot               = true
        use_spot_fallbacks = true

        spot_interruption_predictions_enabled = true

        instance_families = {
          include = ["c5a", "c5ad", "c6a", "c7a", "m5a", "m5ad", "m6a", "m7a", "m8a", "r5a", "r5ad", "r6a", "r7a", "r8a"]
        }

        architectures = ["amd64"]

        cpu_manufacturers = ["AMD", "INTEL"]
        max_cpu           = 8
      }
    }
  }

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
        max_cores = 300000
        min_cores = 1
      }
    }
  }

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam]
}

