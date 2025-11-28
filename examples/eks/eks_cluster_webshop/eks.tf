locals {
  eks_node_group_common = {
    ami_type              = "AL2_x86_64"
    disk_size             = 50
    root_volume_type      = "gp2"
    create_security_group = false
  }
}

#2. create EKS cluster
module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "21.0.0"
  putin_khuylo = true

  name                   = var.cluster_name
  kubernetes_version     = "1.23"
  endpoint_public_access = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  enable_cluster_creator_admin_permissions = true
  authentication_mode                      = "API_AND_CONFIG_MAP"

  addons = {
    coredns    = {}
    kube-proxy = {}
    vpc-cni    = {}
    aws-ebs-csi-driver = {
      service_account_role_arn = module.ebs_csi_irsa_role.iam_role_arn
    }
  }

  self_managed_node_groups = {
    default_node_group = merge(local.eks_node_group_common, {})
    worker-group-1 = merge(local.eks_node_group_common, {
      instance_type = "t3.medium"
      max_size      = 5
      min_size      = 3
      desired_size  = 3
      eni_delete    = "true"
    })
  }

  # Extend cluster security group rules
  security_group_additional_rules = {
    egress_nodes_ephemeral_ports_tcp = {
      description                = "To node 1025-65535"
      protocol                   = "tcp"
      from_port                  = 1025
      to_port                    = 65535
      type                       = "egress"
      source_node_security_group = true
    }
  }

  # Extend node-to-node security group rules
  node_security_group_additional_rules = {
    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }
    egress_all = {
      description      = "Node all egress"
      protocol         = "-1"
      from_port        = 0
      to_port          = 0
      type             = "egress"
      cidr_blocks      = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["::/0"]
    }
    ingress_allow_access_from_control_plane = {
      type                          = "ingress"
      protocol                      = "tcp"
      from_port                     = 9443
      to_port                       = 9443
      source_cluster_security_group = true
      description                   = "Allow access from control plane to webhook port of AWS load balancer controller"
    }
    nginx_ingress_allow_access_from_control_plane = {
      type                          = "ingress"
      protocol                      = "tcp"
      from_port                     = 8443
      to_port                       = 8443
      source_cluster_security_group = true
      description                   = "Allow access from control plane to webhook port of nginx-ingress controller"
    }
  }
}

# Add the CAST AI IAM role which is required for CAST AI nodes to join the cluster.
resource "aws_eks_access_entry" "castai" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Add additional admin role if specified
resource "aws_eks_access_entry" "admin" {
  count         = var.eks_user_role_arn != null ? 1 : 0
  cluster_name  = module.eks.cluster_name
  principal_arn = var.eks_user_role_arn
}

resource "aws_eks_access_policy_association" "admin" {
  count         = var.eks_user_role_arn != null ? 1 : 0
  cluster_name  = module.eks.cluster_name
  principal_arn = var.eks_user_role_arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.admin]
}

module "ebs_csi_irsa_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 4.21.1"

  role_name             = "ebs-csi-${var.cluster_name}"
  attach_ebs_csi_policy = true

  oidc_providers = {
    ex = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

data "aws_eks_cluster" "eks" {
  name = module.eks.cluster_name
}

resource "kubernetes_storage_class" "ebs_csi" {
  metadata {
    name = "ebs-sc"
  }
  storage_provisioner = "ebs.csi.aws.com"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
}
