# 3. Connect EKS cluster to CAST AI with enabled Kvisor security agent.

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

# Install CAST AI with enabled Kvisor security agent.
module "castai-eks-cluster" {
  source = "castai/eks-cluster/castai"

  kvisor_grpc_addr = var.kvisor_grpc_addr

  # Kvisor is an open-source security agent from CAST AI.
  # install_security_agent by default installs Kvisor controller (k8s: deployment)
  # https://docs.cast.ai/docs/kvisor
  install_security_agent = true

  # Kvisor configuration examples, enable certain features:
  kvisor_values = [
    yamlencode({
      controller = {
        extraArgs = {
          # UI: Vulnerability management configuration = API: IMAGE_SCANNING
          "image-scan-enabled" = true
          # UI: Compliance configuration = API: CONFIGURATION_SCANNING
          "kube-bench-enabled"  = true
          "kube-linter-enabled" = true
        }
      }

      # UI: Runtime Security = API: RUNTIME_SECURITY
      agent = {
        # In order to enable Runtime security set agent.enabled to true.
        # This will install Kvisor agent (k8s: daemonset)
        # https://docs.cast.ai/docs/sec-runtime-security
        "enabled" = true

        extraArgs = {
          # Runtime security configuration examples:
          # By default, most users enable the eBPF events and file hash enricher.
          # For all flag explanations and code, see: https://github.com/castai/kvisor/blob/main/cmd/agent/daemon/daemon.go
          "ebpf-events-enabled"        = true
          "file-hash-enricher-enabled" = true
          # other examples
          "netflow-enabled"              = false
          "netflow-export-interval"      = "30s"
          "ebpf-program-metrics-enabled" = false
          "prom-metrics-export-enabled"  = false
          "prom-metrics-export-interval" = "30s"
          "process-tree-enabled"         = false
        }
      }
    })
  ]

  # Deprecated, leave this empty, to prevent setting defaults.
  kvisor_controller_extra_args = {}

  # Everything else...

  wait_for_cluster_ready = false

  install_egressd             = false
  install_workload_autoscaler = false
  install_pod_mutator         = false
  delete_nodes_on_disconnect  = false

  api_url          = var.castai_api_url
  castai_api_token = var.castai_api_token
  grpc_url         = var.castai_grpc_url

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn = module.castai-eks-role-iam.role_arn

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]
  node_configurations = {
    default = {
      subnets = module.vpc.private_subnets
      tags    = {}
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }
  }

  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        spot_interruption_predictions_enabled = false
        spot_interruption_predictions_type    = "aws-rebalance-recommendations"
      }
    }
  }

  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam, module.eks, module.vpc]
}

