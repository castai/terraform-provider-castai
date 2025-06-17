data "aws_availability_zones" "available" {}

data "aws_eks_cluster_auth" "eks_onboarded" {
  name = module.eks.cluster_name
}

# Get the kubernetes service endpoints in the default namespace for Calico installation
data "kubernetes_endpoints_v1" "kubernetes_service" {
  metadata {
    name      = "kubernetes"
    namespace = "default"
  }

  depends_on = [module.eks.cluster_endpoint]
}

provider "aws" {
  region = "eu-central-1" # Set the AWS region to EU Central (Frankfurt)
}

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    token                  = data.aws_eks_cluster_auth.eks_onboarded.token
  }
}

locals {
  vpc_cidr   = "10.0.0.0/16"
  nfs_subnet = "10.0.99.0/24"
  azs        = slice(data.aws_availability_zones.available.names, 0, 3)

  tags = {
    # repo_url = "http://gitlab.com/castai/IaC"
    team      = "live"
    persist   = "true"
    terraform = "true"
  }

  # Create a local value to store the first IP of the kubernetes endpoint -> to install Calico
  all_endpoint_ips = flatten([
    for subset in data.kubernetes_endpoints_v1.kubernetes_service.subset : [
      for addresses in subset : [
        for ip in addresses : ip
      ]
    ]
  ])
  kubernetes_endpoint_ip = length(local.all_endpoint_ips) > 0 ? local.all_endpoint_ips[0].ip : ""
}

# Without that, pods on nodes with Calico don't have network access (internet, nor even node IPs)
resource "aws_security_group_rule" "calico-vxlan" {
  security_group_id = module.eks.node_security_group_id
  type              = "ingress"
  from_port         = 4789
  to_port           = 4789
  protocol          = "udp"
  cidr_blocks       = [local.vpc_cidr]
  description       = "VXLAN calico"
}

# trivy:ignore:aws-ec2-no-excessive-port-access
# trivy:ignore:aws-ec2-no-public-ingress-acl
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = var.cluster_name
  cidr = local.vpc_cidr

  azs             = local.azs
  private_subnets = [for k, v in local.azs : cidrsubnet(local.vpc_cidr, 4, k)]
  public_subnets  = [for k, v in local.azs : cidrsubnet(local.vpc_cidr, 8, k + 48)]

  enable_nat_gateway     = true
  single_nat_gateway     = true
  one_nat_gateway_per_az = false

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }

  tags = local.tags
}

# Security group for VPC endpoints
resource "aws_security_group" "vpc_endpoints_sg" {
  name        = "${var.cluster_name}-vpc-endpoints-sg"
  description = "Security group for VPC endpoints"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description = "HTTPS from VPC"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [local.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }


  tags = local.tags
}

# Add rule to EKS node security group to allow communication with VPC endpoints
resource "aws_security_group_rule" "nodes_to_vpc_endpoints" {
  security_group_id        = module.eks.node_security_group_id
  type                     = "egress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.vpc_endpoints_sg.id
  description              = "Allow nodes to communicate with VPC endpoints"
}

# trivy:ignore:aws-eks-no-public-cluster-access
# trivy:ignore:aws-eks-no-public-cluster-access-to-cidr
# trivy:ignore:aws-ec2-no-public-egress-sgr
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.32"

  cluster_endpoint_public_access = true

  enable_cluster_creator_admin_permissions = true

  enable_irsa = true

  #Do not enable default VPC CNI and kube-proxy as we will install calico
  bootstrap_self_managed_addons = false
  cluster_addons                = {}

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets
  cluster_security_group_additional_rules = {
    allow_all_vpc = {
      type      = "ingress"
      protocol  = "tcp"
      from_port = 0
      to_port   = 0
      cidr_blocks = [
        local.vpc_cidr
      ]
    }
  }

  eks_managed_node_groups = {
    stock_ami = {
      name              = "stock-ami"
      ami_family        = "AmazonLinux2023"
      instance_types    = ["c5a.large"]
      privateNetworking = true
      min_size          = 2
      max_size          = 4
      desired_size      = 2

      iam_role_additional_policies = {
        AmazonSSMManagedInstanceCore = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
      }
    }
  }

  tags = local.tags
}

resource "helm_release" "calico" {
  name = "calico"

  repository = "https://docs.tigera.io/calico/charts"
  chart      = "tigera-operator"
  version    = "3.29.3"

  namespace        = "tigera-operator"
  create_namespace = true

  values = [
    templatefile("${path.module}/calico.yaml", {
      # Trim any quotes and newlines from the IP address
      api_endpoint = trimspace(local.kubernetes_endpoint_ip)
    })
  ]
  wait = false

  depends_on = [data.kubernetes_endpoints_v1.kubernetes_service]
}

resource "null_resource" "deploy_non_blocking_coredns" {
  provisioner "local-exec" {
    command = "aws eks create-addon --cluster-name ${var.cluster_name} --region ${var.region} --addon-name coredns --addon-version v1.11.4-eksbuild.2"
  }
  depends_on = [module.eks]
}