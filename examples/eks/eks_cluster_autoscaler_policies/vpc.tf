# 1. Create VPC with hyperscale network configuration
# This configuration provides large-scale network capacity:
# - 24 private subnets across 6 AZs (4 subnets per AZ)
# - /21 CIDR blocks per subnet (2,048 IPs each = ~49K total private IPs)
# - VPC CIDR: 100.64.0.0/16 (supports 65,536 IPs)

data "aws_availability_zones" "available" {}

locals {
  # Use all 6 availability zones in us-east-1
  azs = [
    "us-east-1a", # use1-az4
    "us-east-1b", # use1-az6
    "us-east-1c", # use1-az1
    "us-east-1d", # use1-az2
    "us-east-1e", # use1-az3
    "us-east-1f", # use1-az5
  ]

  # 24 private subnets: 4 per AZ with /21 CIDR blocks (2,048 IPs each)
  # Distribution: 100.64.0.0/21 through 100.64.184.0/21
  private_subnets = [
    # AZ-a (us-east-1a) - 4 subnets
    "100.64.0.0/21",   # 100.64.0.0 - 100.64.7.255
    "100.64.8.0/21",   # 100.64.8.0 - 100.64.15.255
    "100.64.16.0/21",  # 100.64.16.0 - 100.64.23.255
    "100.64.24.0/21",  # 100.64.24.0 - 100.64.31.255

    # AZ-b (us-east-1b) - 4 subnets
    "100.64.32.0/21",  # 100.64.32.0 - 100.64.39.255
    "100.64.40.0/21",  # 100.64.40.0 - 100.64.47.255
    "100.64.48.0/21",  # 100.64.48.0 - 100.64.55.255
    "100.64.56.0/21",  # 100.64.56.0 - 100.64.63.255

    # AZ-c (us-east-1c) - 4 subnets
    "100.64.64.0/21",  # 100.64.64.0 - 100.64.71.255
    "100.64.72.0/21",  # 100.64.72.0 - 100.64.79.255
    "100.64.80.0/21",  # 100.64.80.0 - 100.64.87.255
    "100.64.88.0/21",  # 100.64.88.0 - 100.64.95.255

    # AZ-d (us-east-1d) - 4 subnets
    "100.64.96.0/21",   # 100.64.96.0 - 100.64.103.255
    "100.64.104.0/21",  # 100.64.104.0 - 100.64.111.255
    "100.64.112.0/21",  # 100.64.112.0 - 100.64.119.255
    "100.64.120.0/21",  # 100.64.120.0 - 100.64.127.255

    # AZ-e (us-east-1e) - 4 subnets
    "100.64.128.0/21",  # 100.64.128.0 - 100.64.135.255
    "100.64.136.0/21",  # 100.64.136.0 - 100.64.143.255
    "100.64.144.0/21",  # 100.64.144.0 - 100.64.151.255
    "100.64.152.0/21",  # 100.64.152.0 - 100.64.159.255

    # AZ-f (us-east-1f) - 4 subnets
    "100.64.160.0/21",  # 100.64.160.0 - 100.64.167.255
    "100.64.168.0/21",  # 100.64.168.0 - 100.64.175.255
    "100.64.176.0/21",  # 100.64.176.0 - 100.64.183.255
    "100.64.184.0/21",  # 100.64.184.0 - 100.64.191.255
  ]

  # 6 public subnets: 1 per AZ with /24 CIDR blocks (256 IPs each)
  public_subnets = [
    "100.64.192.0/24",  # us-east-1a
    "100.64.193.0/24",  # us-east-1b
    "100.64.194.0/24",  # us-east-1c
    "100.64.195.0/24",  # us-east-1d
    "100.64.196.0/24",  # us-east-1e
    "100.64.197.0/24",  # us-east-1f
  ]
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = var.cluster_name
  cidr = "100.64.0.0/16" # Supports 65,536 IPs

  azs             = local.azs
  private_subnets = local.private_subnets
  public_subnets  = local.public_subnets

  # Enable NAT gateway for private subnets
  enable_nat_gateway     = true
  single_nat_gateway     = false # Use one NAT gateway per AZ for HA
  one_nat_gateway_per_az = true

  # Enable DNS
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
