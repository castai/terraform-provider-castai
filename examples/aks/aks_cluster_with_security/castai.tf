# 3. Connect AKS cluster to CAST AI with enabled Kvisor security agent.

# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module with enabled Kvisor security agent.
module "castai-aks-cluster" {
  source = "castai/aks/castai"

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

  install_workload_autoscaler = false
  install_pod_mutator         = false
  delete_nodes_on_disconnect  = var.delete_nodes_on_disconnect

  api_url          = var.castai_api_url
  castai_api_token = var.castai_api_token
  grpc_url         = var.castai_grpc_url

  aks_cluster_name    = var.cluster_name
  aks_cluster_region  = var.cluster_region
  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [azurerm_subnet.internal.id]
      tags           = var.tags
    }
  }
}
