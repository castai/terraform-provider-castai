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

data "aws_eks_cluster_auth" "eks" {
  name = module.eks.cluster_id
}

data "castai_eks_clusterid" "castai_cluster_id" {
  account_id                 = var.aws_account_id
  region                     = var.cluster_region
  cluster_name               = var.cluster_name
}

data "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = data.castai_eks_clusterid.castai_cluster_id.id
}

module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn    = data.castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_id

  // You can provide SGs that CAST AI should use
  override_security_groups = null
  aws_assume_role_arn           = module.castai-eks-role-iam.role_arn
  aws_instance_profile_arn      = module.castai-eks-role-iam.instance_profile_arn

  subnets         = var.subnets
  tags            = var.tags

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  # Full schema can be found here https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies
  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "isScopedMode": false,
        "unschedulablePods": {
            "enabled": true
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["aws"],
            "spotBackups": {
                "enabled": true
            }
        },
        "nodeDownscaler": {
            "emptyNodes": {
                "enabled": true
            }
        }
    }
  EOT

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam"
  depends_on = [module.castai-eks-role-iam]
}
