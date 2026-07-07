# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

data "azurerm_kubernetes_cluster" "example" {
  name                = var.cluster_name
  resource_group_name = var.resource_group
}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module with Workload Identity.
# Workload Identity uses federated credentials instead of App Registration client secrets,
# eliminating secret rotation and scoping permissions to the cluster's resource groups.
module "castai_aks_cluster" {
  source  = "castai/aks/castai"
  version = "~> 10.3"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aks_cluster_name           = var.cluster_name
  aks_cluster_region         = var.cluster_region
  node_resource_group        = data.azurerm_kubernetes_cluster.example.node_resource_group
  resource_group             = data.azurerm_kubernetes_cluster.example.resource_group_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  authentication_method = "workload_identity"

  default_node_configuration = module.castai_aks_cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      min_disk_size  = 100
      disk_cpu_ratio = 0
      subnets        = var.subnets
      tags           = var.tags
    }
  }

  autoscaler_settings = {
    enabled                                 = false
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = false
    }

    node_downscaler = {
      enabled = false

      empty_nodes = {
        enabled = false
      }

      evictor = {
        aggressive_mode           = false
        cycle_interval            = "60s"
        dry_run                   = false
        enabled                   = false
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = false

      cpu = {
        max_cores = 200
        min_cores = 1
      }
    }
  }
}
