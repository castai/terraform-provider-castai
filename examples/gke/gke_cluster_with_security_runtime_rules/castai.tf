# 3. Connect GKE cluster to CAST AI with enabled Kvisor security agent.

module "castai-gke-iam" {
  source = "castai/gke-iam/castai"

  project_id                  = var.project_id
  gke_cluster_name            = var.cluster_name
  service_accounts_unique_ids = length(var.service_accounts_unique_ids) == 0 ? [] : var.service_accounts_unique_ids
}

# Configure GKE cluster connection to CAST AI with enabled Kvisor security agent.
module "castai-gke-cluster" {
  source = "castai/gke-cluster/castai"

  wait_for_cluster_ready = true
  kvisor_grpc_addr       = var.kvisor_grpc_addr

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

  # Everything else ...

  install_workload_autoscaler = false
  install_cloud_proxy         = false
  install_pod_mutator         = false
  delete_nodes_on_disconnect  = false

  api_url          = var.castai_api_url
  castai_api_token = var.castai_api_token
  grpc_url         = var.castai_grpc_url

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = var.cluster_region

  gke_credentials            = module.castai-gke-iam.private_key
  default_node_configuration = module.castai-gke-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      min_disk_size  = 100
      disk_cpu_ratio = 0
      subnets        = [module.vpc.subnets_ids[0]]
      tags           = {}
    }
  }

  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-gke-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand = true
      }
    }
  }

  depends_on = [google_container_cluster.my-k8s-cluster, module.castai-gke-iam]
}