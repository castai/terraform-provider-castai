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
  interface_endpoints = [
    "ecr.api",
    "ecr.dkr",
    "logs",
    "sts"
  ]
}

resource "aws_vpc_endpoint" "interface_endpoints" {
  for_each = toset(local.interface_endpoints)

  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.cluster_region}.${each.key}"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  private_dns_enabled = true
  security_group_ids  = [aws_security_group.vpce_sg.id]
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id          = module.vpc.vpc_id
  service_name    = "com.amazonaws.${var.cluster_region}.s3"
  route_table_ids = module.vpc.private_route_table_ids
}

resource "aws_security_group" "vpce_sg" {
  name        = "vpc-endpoints-sg"
  description = "Allow access to VPC interface endpoints"
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
}

resource "aws_vpc_endpoint" "cast_ai_rest_api" {
  vpc_id              = module.vpc.vpc_id
  service_name        = var.rest_api_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.cast_ai_vpc_service.id]
  private_dns_enabled = true

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


