# 2. Create EKS cluster.
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name                                     = var.cluster_name
  kubernetes_version                       = var.cluster_version
  endpoint_public_access                   = true
  enable_cluster_creator_admin_permissions = true

  addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent    = true
      before_compute = true
    }
    aws-ebs-csi-driver = {
      service_account_role_arn = module.ebs_csi_irsa_role.iam_role_arn
      most_recent              = true
      resolve_conflicts        = "OVERWRITE"
    }
  }

  tags = var.tags

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Access entry for CAST AI nodes to join the cluster
  access_entries = {
    castai_node = {
      principal_arn = module.castai-eks-role-iam[0].instance_profile_role_arn
      type          = "EC2_LINUX"
    }
  }

  self_managed_node_groups = {
    node_group_1 = {
      name          = "${var.cluster_name}-ng-1"
      instance_type = "m5.large"
      max_size      = 5
      min_size      = 2
      desired_size  = 2

      # Allow pods to access IMDS (required for castai-agent)
      metadata_options = {
        http_endpoint               = "enabled"
        http_tokens                 = "required"
        http_put_response_hop_limit = 2
      }
    }
  }

  eks_managed_node_groups = {
    node_group_spot = {
      name         = "${var.cluster_name}-spot"
      min_size     = 1
      max_size     = 10
      desired_size = 1

      metadata_options = {
        http_endpoint               = "enabled"
        http_tokens                 = "required"
        http_put_response_hop_limit = 2
      }

      instance_types = ["t3.large"]
      capacity_type  = "SPOT"

      update_config = {
        max_unavailable_percentage = 50 # or set `max_unavailable`
      }
    }
  }

  node_security_group_additional_rules = {
    # Match eksctl default behaviour: allow the control plane to reach all workload TCP ports on nodes
    # (ports 1025-65535). This covers any admission webhook on any port (e.g. castai-live on 9091,
    # Istio on 15017/15012, etc.) without needing to enumerate each one individually.
    # See: https://github.com/eksctl-io/eksctl/blob/main/pkg/cfn/builder/nodegroup.go
    #      ControlPlaneNodeGroupEgressRules / makeNodeIngressRules
    ingress_cluster_all_workload_ports = {
      description                   = "Cluster API to node workload TCP ports (kubelet and webhooks)"
      protocol                      = "tcp"
      from_port                     = 1025
      to_port                       = 65535
      type                          = "ingress"
      source_cluster_security_group = true
    }
  }

}

module "ebs_csi_irsa_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.0"

  role_name             = "ebs-csi-${var.cluster_name}"
  attach_ebs_csi_policy = true

  oidc_providers = {
    ex = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

# The gp2 StorageClass is created by the EKS control plane / aws-ebs-csi-driver addon
# asynchronously after the EKS module completes. A short sleep gives the addon time to
# reconcile before we attempt to patch the annotation.
resource "time_sleep" "wait_for_gp2" {
  create_duration = "30s"
  depends_on      = [module.eks]
}

# Mark the EKS-created gp2 StorageClass as the cluster default.
# EKS ships gp2 without the default annotation, so PVCs without an explicit
# storageClassName would fail. We patch it here rather than creating a new SC.
resource "kubernetes_annotations" "gp2_default_storage_class" {
  api_version = "storage.k8s.io/v1"
  kind        = "StorageClass"
  metadata {
    name = "gp2"
  }
  annotations = {
    "storageclass.kubernetes.io/is-default-class" = "true"
  }

  depends_on = [time_sleep.wait_for_gp2]
}

# Example additional security group.
resource "aws_security_group" "additional" {
  name_prefix = "${var.cluster_name}-additional"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"
    cidr_blocks = [
      "10.0.0.0/8",
    ]
  }
}
