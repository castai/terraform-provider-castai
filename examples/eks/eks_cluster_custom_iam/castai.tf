# 4. Connect EKS cluster to CAST AI.

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

data "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

provider "castai" {
  api_url = var.castai_api_url
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
        args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
      }
  }
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

  aws_account_id      = data.aws_caller_identity.current.account_id
  aws_cluster_region  = var.cluster_region
  aws_cluster_name    = module.eks.cluster_name
  aws_assume_role_arn = aws_iam_role.assume_role.arn

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  # Define CAST AI default node configuration.
  node_configurations = {
    default = {
      subnets         = module.vpc.private_subnets
      tags            = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = aws_iam_instance_profile.castai_instance_profile.arn
    }
  }

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}
