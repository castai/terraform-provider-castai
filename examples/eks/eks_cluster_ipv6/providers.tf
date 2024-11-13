# Following providers required by EKS and VPC modules.
provider "aws" {
  region = var.cluster_region
}

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  # Make sure awscli is installed locally where Terraform is executed.
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.cluster_region]
  }
}
