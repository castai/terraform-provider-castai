# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.cluster_name # Replace with the actual name of your EKS cluster
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

locals {
  access_entry = can(regex("API", data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode))
}

resource "aws_eks_access_entry" "access_entry" {
  count         = local.access_entry ? 1 : 0
  cluster_name  = data.aws_eks_cluster.existing_cluster.name
  principal_arn = module.castai_eks_role_iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai_eks_role_iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = data.aws_eks_cluster.existing_cluster.vpc_config[0].vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

module "castai_eks_cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 17.0" # Storage Autoscaler support (install_storage_autoscaler) was added in v17.0.0

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  kvisor_grpc_addr       = var.kvisor_grpc_addr
  wait_for_cluster_ready = true

  # Required for umbrella installations. Keeps Workload Autoscaler CRDs if standalone
  # charts are removed in favor of the umbrella chart. Existing standalone installations
  # must apply this with umbrella_enabled=false first, then enable the umbrella separately.
  workload_autoscaler_keep_crds = true
  umbrella_enabled              = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = module.castai_eks_role_iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai_eks_cluster.castai_node_configurations["default"]
  node_configurations = {
    default = {
      subnets = data.aws_eks_cluster.existing_cluster.vpc_config[0].subnet_ids
      tags    = var.tags
      security_groups = concat(
        [data.aws_eks_cluster.existing_cluster.vpc_config[0].cluster_security_group_id],
        tolist(data.aws_eks_cluster.existing_cluster.vpc_config[0].security_group_ids)
      )
      instance_profile_arn = module.castai_eks_role_iam.instance_profile_arn
    }
  }

  # Installs Storage Autoscaler sub-chart of the Cast AI umbrella Helm chart.
  # Installation starts in dry-run mode: planned actions are logged, no PVCs are modified.
  install_storage_autoscaler = true

  # Binds the IRSA role from iam.tf to the `castai-storage-autoscaler`
  # Kubernetes service account so the controller can call AWS APIs.
  storage_autoscaler_values = [
    yamlencode({
      config = {
        # Set to false to let Storage Autoscaler apply changes to PVCs (live mode).
        # See https://docs.cast.ai/docs/storage-installation#enable-live-mode
        dryRun = true
      }
      serviceAccount = {
        name = local.storage_autoscaler_service_account
        annotations = {
          "eks.amazonaws.com/role-arn" = aws_iam_role.castai_storage_autoscaler.arn
        }
      }
    })
  ]

  # Installs Kvisor (https://docs.cast.ai/docs/kvisor). Its storage metrics populate the
  # Storage Autoscaler dashboard in the Cast AI console: savings estimates, storage type
  # breakdown, resize event logs and per-PV usage. Storage Autoscaler works without Kvisor,
  # but the dashboard stays empty.
  install_security_agent = true

  # See https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics
  kvisor_values = [
    yamlencode({
      controller = {
        # Storage metrics collection requires both the Kvisor controller and agent.
        enabled = true

        extraArgs = {
          # Collect cloud volume metadata (disk type, size, provisioned IOPS) from the EC2 API.
          "cloud-provider"                      = "aws"
          "cloud-provider-aws-region"           = var.cluster_region
          "cloud-provider-storage-sync-enabled" = true
        }

        # Binds the IRSA role from iam.tf (ec2:DescribeVolumes) to the Kvisor controller.
        serviceAccount = {
          name = local.kvisor_controller_service_account
          annotations = {
            "eks.amazonaws.com/role-arn" = aws_iam_role.castai_kvisor.arn
          }
        }
      }

      agent = {
        enabled = true

        extraArgs = {
          # Collect ephemeral storage and persistent volume usage per node.
          "storage-stats-enabled" = true
        }
      }
    })
  ]

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai_eks_role_iam]
}
