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

module "castai-aws-iam" {
  source = "../../../terraform-castai-eks"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  create_iam_resources_per_cluster = true
}

resource "helm_release" "castai_agent" {
  name            = "castai-agent"
  repository      = "https://castai.github.io/helm-charts"
  chart           = "castai-agent"
  cleanup_on_fail = true

  set {
    name  = "provider"
    value = "eks"
  }

  set_sensitive {
    name  = "apiKey"
    value = castai_eks_cluster.my_castai_cluster.cluster_token
  }
}

resource "castai_eks_cluster" "my_castai_cluster" {
  account_id = var.aws_account_id
  region     = var.cluster_region
  name       = module.eks.cluster_id

  access_key_id        = module.castai-aws-iam.aws_access_key_id
  secret_access_key    = module.castai-aws-iam.aws_secret_access_key
  instance_profile_arn = module.castai-aws-iam.instance_profile_role_arn

  depends_on = [module.eks]
}

resource "helm_release" "castai_cluster_controller" {
  name            = "castai-cluster-controller"
  repository      = "https://castai.github.io/helm-charts"
  chart           = "castai-cluster-controller"
  cleanup_on_fail = true

  set {
    name = "castai.clusterID"
    value = castai_eks_cluster.my_castai_cluster.id
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_eks_cluster.my_castai_cluster.cluster_token
  }
}
