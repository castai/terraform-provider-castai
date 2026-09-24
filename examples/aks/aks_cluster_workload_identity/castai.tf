# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

data "azurerm_kubernetes_cluster" "example" {
  name                = var.cluster_name
  resource_group_name = var.resource_group
}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module.
# authentication_method = "workload_identity" makes the module use a User-Assigned
# Managed Identity with a federated identity credential instead of an App
# Registration with a client secret.
module "castai_aks_cluster" {
  source  = "castai/aks/castai"
  version = "~> 12.0"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  workload_autoscaler_keep_crds = true
  umbrella_enabled              = true

  authentication_method = "workload_identity"

  aks_cluster_name           = var.cluster_name
  aks_cluster_region         = var.cluster_region
  node_resource_group        = data.azurerm_kubernetes_cluster.example.node_resource_group
  resource_group             = data.azurerm_kubernetes_cluster.example.resource_group_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  # Use name-based references (default_node_configuration_name / configuration_name)
  # instead of module.castai_aks_cluster outputs: a module block cannot reference
  # its own outputs, Terraform would reject the configuration with a dependency
  # cycle ("Self-referential module block").
  default_node_configuration_name = "default"
  install_workload_autoscaler     = true

  node_configurations = {
    default = {
      min_disk_size  = 100
      disk_cpu_ratio = 0
      subnets        = var.subnets
      tags           = var.tags
    }
  }

  node_templates = {
    default_by_castai = {
      name               = "default-by-castai"
      configuration_name = "default"
      is_default         = true
      is_enabled         = true
      should_taint       = false

      constraints = {
        on_demand = true
      }
    }

    example_spot_template = {
      configuration_name = "default"
      is_enabled         = true
      should_taint       = true

      custom_labels = {
        custom-label-key-1 = "custom-label-value-1"
        custom-label-key-2 = "custom-label-value-2"
      }

      custom_taints = [
        {
          key    = "custom-taint-key-1"
          value  = "custom-taint-value-1"
          effect = "NoSchedule"
        },
        {
          key    = "custom-taint-key-2"
          value  = "custom-taint-value-2"
          effect = "NoSchedule"
        }
      ]

      constraints = {
        spot                          = true
        use_spot_fallbacks            = true
        fallback_restore_rate_seconds = 1800
        min_cpu                       = 4
        max_cpu                       = 100
        instance_families = {
          exclude = ["standard_FSv2"]
        }
        custom_priority = {
          instance_families = ["standard_Dv4"]
          spot              = true
        }
      }
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

  install_omni = true
}
