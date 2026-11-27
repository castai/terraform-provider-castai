locals {
  # Passes --profile to `aws eks get-token` only when var.profile is set. Without it, the AWS CLI
  # falls back to the same credential chain as the AWS provider (AWS_PROFILE, env keys, SSO, default).
  aws_cli_profile_args = var.profile != null ? ["--profile", var.profile] : []
}

# Following providers required by the CAST AI eks-cluster module.
provider "aws" {
  region = var.cluster_region
  # null lets the provider use the standard AWS credential chain, including AWS_PROFILE.
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
    args = concat(["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region], local.aws_cli_profile_args)
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
      args = concat(["eks", "get-token", "--cluster-name", var.cluster_name, "--region", var.cluster_region], local.aws_cli_profile_args)
    }
  }
}
