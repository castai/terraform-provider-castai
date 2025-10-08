#1. Create VPC.
data "aws_availability_zones" "available" {}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = var.cluster_name
  cidr = "10.0.0.0/16"

  azs                  = data.aws_availability_zones.available.names
  private_subnets      = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets       = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"           = 1
  }
}

resource "aws_network_acl" "private_egress_control" {
  vpc_id = module.vpc.vpc_id

  ingress {
    protocol   = "-1"
    rule_no    = 100
    action     = "allow"
    cidr_block = module.vpc.vpc_cidr_block
    from_port  = 0
    to_port    = 0
  }

  dynamic "egress" {
    for_each = var.gcp_api_ip_ranges
    content {
      protocol   = "tcp"
      rule_no    = 100 + index(var.gcp_api_ip_ranges, egress.value) # care to avoid collisions
      action     = "allow"
      cidr_block = egress.value
      from_port  = 443
      to_port    = 443
    }
  }

  egress {
    protocol   = "tcp"
    rule_no    = 200
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 1024
    to_port    = 65535
  }

  egress {
    protocol   = "-1"
    rule_no    = 300
    action     = "deny"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags = {
    Name = "private-egress-control"
  }
}

resource "aws_network_acl_association" "private_subnet_assoc" {
  for_each = toset(module.vpc.private_subnets)

  subnet_id      = each.key
  network_acl_id = aws_network_acl.private_egress_control.id
}

