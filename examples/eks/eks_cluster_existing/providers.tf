# Following providers required by EKS and VPC modules.
provider "aws" {
  region  = var.cluster_region
  profile = var.profile
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.existing_cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.existing_cluster.certificate_authority.0.data)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    # This requires the awscli to be installed locally where Terraform is executed
    args = ["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region, "--profile", var.profile]
  }
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
