data "aws_availability_zones" "available" {}

locals {
  azs = [
    "us-east-1a",
    "us-east-1b",
    "us-east-1c",
    "us-east-1d",
    "us-east-1f",
  ]

  private_subnets = [
    "100.64.0.0/21",
    "100.64.8.0/21",
    "100.64.16.0/21",
    "100.64.24.0/21",
    "100.64.128.0/21",
    "100.64.32.0/21",
    "100.64.40.0/21",
    "100.64.48.0/21",
    "100.64.56.0/21",
    "100.64.136.0/21",
    "100.64.64.0/21",
    "100.64.72.0/21",
    "100.64.80.0/21",
    "100.64.88.0/21",
    "100.64.144.0/21",
    "100.64.96.0/21",
    "100.64.104.0/21",
    "100.64.112.0/21",
    "100.64.120.0/21",
    "100.64.152.0/21",
    "100.64.160.0/21",
    "100.64.168.0/21",
    "100.64.176.0/21",
    "100.64.184.0/21",
  ]

  public_subnets = [
    "100.64.192.0/24",
    "100.64.193.0/24",
    "100.64.194.0/24",
    "100.64.195.0/24",
    "100.64.197.0/24",
  ]
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = var.cluster_name
  cidr = "100.64.0.0/16"

  azs             = local.azs
  private_subnets = local.private_subnets
  public_subnets  = local.public_subnets

  enable_nat_gateway     = true
  single_nat_gateway     = false
  one_nat_gateway_per_az = true

  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "sys-environment"                           = "dev"
    "sys-team"                                  = "hpc"
  }

  public_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                    = 1
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"           = 1
    "sys-type"                                  = "compute"
  }
}
