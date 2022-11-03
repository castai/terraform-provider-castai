#2. create EKS cluster
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "17.24.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.22"

  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.private_subnets

  cluster_endpoint_private_access                = true
  cluster_create_endpoint_private_access_sg_rule = true
  cluster_endpoint_private_access_cidrs          = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]

  node_groups_defaults = {
    ami_type  = "AL2_x86_64"
    disk_size = 50
  }

  workers_group_defaults = {
    root_volume_type = "gp2"
  }

  worker_additional_security_group_ids = [aws_security_group.all_worker_mgmt.id]

  worker_groups = [
    {
      name                 = "worker-group-1"
      instance_type        = "t3.medium"
      asg_desired_capacity = 1
      additional_security_group_ids = [
        aws_security_group.worker_group_mgmt_one.id, aws_security_group.worker_group_mgmt_two.id
      ]
      eni_delete = true
    },
  ]

  map_roles = [
    # ADD - give access to nodes spawned by cast.ai
    {
      rolearn  = module.castai-eks-role-iam.instance_profile_role_arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups   = ["system:bootstrappers", "system:nodes"]
    },
  ]
}

data "aws_eks_cluster" "eks" {
  name = module.eks.cluster_id
}
