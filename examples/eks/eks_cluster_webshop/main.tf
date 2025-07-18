locals {
  role_name = "castai-eks-role-${var.cluster_name}"
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.eks.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.eks.token
  }
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "aws" {
  region     = var.cluster_region
  access_key = var.aws_access_key_id
  secret_key = var.aws_secret_access_key
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks.token
}

provider "kubectl" {
  host                   = data.aws_eks_cluster.eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks.token
}

data "aws_caller_identity" "current" {}

data "aws_eks_cluster_auth" "eks" {
  name = module.eks.cluster_id
}

resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 1.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 12.0"

  api_url = var.castai_api_url

  aws_account_id      = data.aws_caller_identity.current.account_id
  aws_cluster_region  = var.cluster_region
  aws_cluster_name    = module.eks.cluster_id
  aws_assume_role_arn = module.castai-eks-role-iam.role_arn

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.worker_group_mgmt_one.id,
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
        spot               = true
        use_spot_fallbacks = true

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20
      }
    }
  }

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect


  autoscaler_settings = {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true
    }

    node_downscaler = {
      empty_nodes = {
        enabled = true
      }
    }
  }

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam"
  depends_on = [module.castai-eks-role-iam]
}
