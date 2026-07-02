# Following providers required by EKS and CAST AI modules.
provider "aws" {
  region = var.cluster_region
}

provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.existing_cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.existing_cluster.certificate_authority.0.data)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    # This requires the awscli to be installed locally where Terraform is executed
    args = ["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region]
  }
}

provider "helm" {
  kubernetes = {
    host                   = data.aws_eks_cluster.existing_cluster.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.existing_cluster.certificate_authority.0.data)
    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      # This requires the awscli to be installed locally where Terraform is executed.
      args = ["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region]
    }
  }
}
