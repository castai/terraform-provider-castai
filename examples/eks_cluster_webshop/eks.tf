#2. create EKS cluster
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "18.20.5"

  cluster_name    = var.cluster_name
  cluster_version = "1.22"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_private_access                = true

  eks_managed_node_group_defaults = {
    ami_type  = "AL2_x86_64"
    disk_size = 50
  }

  self_managed_node_group_defaults = {
    root_volume_type = "gp2"
    create_security_group = false
  }
  cluster_addons = {
    coredns = {
      resolve_conflicts = "OVERWRITE"
    }
    kube-proxy = {}
    vpc-cni = {
      resolve_conflicts = "OVERWRITE"
    }
    aws-ebs-csi-driver = {}
  }

  self_managed_node_groups = {
    default_node_group = {}
    worker-group-1 = {
      instance_type        = "t3.medium"
      asg_desired_capacity = 3
      eni_delete = "true"
    },
  }

  # Extend cluster security group rules
  cluster_security_group_additional_rules = {
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
  }

  create_aws_auth_configmap = true
  manage_aws_auth_configmap = true

  aws_auth_roles = [
    # ADD - give access to nodes spawned by cast.ai
    {
      rolearn  = module.castai-eks-role-iam.instance_profile_role_arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups   = ["system:bootstrappers", "system:nodes"]
    },
  ]
}

module "ebs_csi_irsa_role" {
  source = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 4.21.1"

  role_name             = "ebs-csi"
  attach_ebs_csi_policy = true

  oidc_providers = {
    ex = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

data "aws_eks_cluster" "eks" {
  name = module.eks.cluster_id
}

