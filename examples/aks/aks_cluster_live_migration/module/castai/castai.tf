module "castai-aks-cluster" {
  source  = "castai/aks/castai"
  version = "~> 10.3"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aks_cluster_name    = var.cluster_name
  aks_cluster_region  = var.cluster_region
  node_resource_group = var.node_resource_group
  resource_group      = var.resource_group

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id

  default_node_configuration_name = "default"

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [var.subnet_id]
      tags           = var.tags
    }

    live = {
      disk_cpu_ratio = 25
      subnets        = [var.subnet_id]
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
        on_demand          = true
        spot               = true
        use_spot_fallbacks = true

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20
      }
    }

    live_tmpl = {
      configuration_name = "live"
      is_enabled         = true
      should_taint       = true
      clm_enabled        = true

      constraints = {
        instance_families = {
          exclude = []
          include = ["standard_dsv2", "standard_dsv3", "standard_dsv4", "standard_dsv5", "standard_ev3", "standard_ev4"]
        }
      }
    }
  }

  autoscaler_settings = {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true
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

  install_live = var.install_helm_live
  live_version = var.install_helm_live ? var.live_helm_version : null
}
