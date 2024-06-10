# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

data "azurerm_kubernetes_cluster" "example" {
  name                = var.cluster_name
  resource_group_name = var.cluster_rg
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = data.azurerm_kubernetes_cluster.example.kube_config.0.host
    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_certificate)
    client_key             = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_key)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.cluster_ca_certificate)
  }
}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module.
module "castai-aks-cluster" {
  source = "castai/aks/castai"

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


  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio    = 0
      subnets           = var.subnets
      tags              = var.tags
      max_pods_per_node = 60
    }
  }

  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-aks-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand  = true
        min_cpu    = 8
        max_cpu    = 96
        max_memory = 786432
        instance_families = {
          exclude = ["standard_FSv2", "standard_Dv4"]
        }
      }
    }
  }

  autoscaler_policy_overrides = {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true

      headroom = {
        enabled           = true
        cpu_percentage    = 10
        memory_percentage = 10
      }

      headroom_spot = {
        enabled           = true
        cpu_percentage    = 10
        memory_percentage = 10
      }
    }

    node_downscaler = {
      enabled = true

      empty_nodes = {
        enabled = true
      }

      evictor = {
        aggressive_mode           = false
        cycle_interval            = "5m10s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = true

      cpu = {
        max_cores = 20
        min_cores = 1
      }
    }
  }

}


resource "castai_rebalancing_schedule" "default" {
  name = "rebalance nodes at every 30th minute"
  schedule {
    cron = "CRON_TZ=America/Argentina/Buenos_Aires */30 * * * *"
  }
  trigger_conditions {
    savings_percentage = 20
  }
  launch_configuration {
    # only consider instances older than 5 minutes
    node_ttl_seconds         = 300
    num_targeted_nodes       = 3
    rebalancing_min_nodes    = 2
    keep_drain_timeout_nodes = false
    execution_conditions {
      enabled                     = true
      achieved_savings_percentage = 10
    }
  }
}

resource "castai_rebalancing_job" "default" {
  cluster_id              = module.castai-aks-cluster.cluster_id
  rebalancing_schedule_id = castai_rebalancing_schedule.default.id
  enabled                 = true
}
