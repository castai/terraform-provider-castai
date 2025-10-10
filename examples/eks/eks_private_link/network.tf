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
  enable_dns_hostnames = true
  enable_dns_support   = true
  enable_nat_gateway   = true
  single_nat_gateway   = true


  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"           = 1
    "cast.ai/routable"                          = "true"
  }
}


resource "aws_security_group" "cast_ai_vpc_service" {
  name   = "SG used by NGINX proxy VMs"
  vpc_id = module.vpc.vpc_id

  ingress {
    description      = "Accessing CAST AI endpoints"
    from_port        = 443
    to_port          = 443
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  depends_on = [
    module.vpc
  ]
}

locals {
  gateway_endpoints = [
    "s3"
  ]
}

resource "aws_vpc_endpoint" "gateway" {
  for_each          = toset(local.gateway_endpoints)
  vpc_id            = module.vpc.vpc_id
  service_name      = "com.amazonaws.${var.cluster_region}.${each.value}"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = module.vpc.private_route_table_ids
  tags = {
    Name = "${var.cluster_name}-${each.value}-vpce"
  }

  depends_on = [
    module.vpc
  ]
}

locals {
  interface_endpoints = [
    "ec2",
    "ec2messages",
    "ssm",
    "ssmmessages",
    "monitoring",
    "logs",
    "ecr.api",
    "ecr.dkr",
    "secretsmanager",
    "sts",
    "ecs-agent",
    "ecs-telemetry"
  ]
}

resource "aws_vpc_endpoint" "interface" {
  for_each            = toset(local.interface_endpoints)
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.cluster_region}.${each.value}"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoint_sg.id]
  private_dns_enabled = true
  tags = {
    Name = "${var.cluster_name}-${each.value}-vpce"
  }

  depends_on = [
    module.vpc
  ]
}


resource "aws_security_group" "vpc_endpoint_sg" {
  name        = "${var.cluster_name}-vpce-sg"
  description = "SG for VPC interface endpoints"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [module.vpc.vpc_cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.cluster_name}-vpce-sg"
  }
}

resource "aws_vpc_endpoint" "cast_ai_rest_api" {
  vpc_id             = module.vpc.vpc_id
  service_name       = var.rest_api_service_name
  vpc_endpoint_type  = "Interface"
  subnet_ids         = module.vpc.private_subnets
  security_group_ids = [aws_security_group.cast_ai_vpc_service.id]

  depends_on = [
    module.vpc
  ]
}

resource "aws_vpc_endpoint" "cast_ai_grpc" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.grpc_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

  depends_on = [
    module.vpc
  ]
}

resource "aws_vpc_endpoint" "cast_ai_api_grpc" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.api_grpc_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

  depends_on = [
    module.vpc
  ]
}

resource "aws_vpc_endpoint" "cast_ai_files" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.files_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

  depends_on = [
    module.vpc
  ]
}

resource "aws_vpc_endpoint" "cast_ai_kvisor" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.kvisor_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

  depends_on = [
    module.vpc
  ]
}

resource "aws_vpc_endpoint" "cast_ai_telemetry" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.telemetry_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

  depends_on = [
    module.vpc
  ]
}

