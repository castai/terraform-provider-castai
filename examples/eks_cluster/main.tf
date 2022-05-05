provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.eks.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.eks.token
  }
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks.token
}

data "aws_eks_cluster_auth" "eks" {
  name = module.eks.cluster_id
}

provider "castai" {
  api_token = var.castai_api_token
}

provider "aws" {
  region     = var.cluster_region
  access_key = var.aws_access_key_id
  secret_key = var.aws_secret_access_key
}

module "castai-eks-iam" {
  source  = "castai/eks-iam/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  create_iam_resources_per_cluster = true
}

module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_id

  aws_access_key_id             = module.castai-eks-iam.aws_access_key_id
  aws_secret_access_key         = module.castai-eks-iam.aws_secret_access_key
  aws_instance_profile_arn      = module.castai-eks-iam.instance_profile_arn

  subnets         = var.subnets
  override_security_groups = var.security_groups
  tags            = var.tags

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  ssh_public_key = "key-123"

  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "isScopedMode": true,
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
}
