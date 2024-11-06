resource "castai_aks_cluster" "this" {
  name = var.cluster_name

  region          = var.cluster_region
  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id
  client_id       = azuread_application.castai.application_id
  client_secret   = azuread_application_password.castai.value


  node_resource_group        = data.azurerm_kubernetes_cluster.example.node_resource_group
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}

resource "castai_node_configuration" "default" {
  cluster_id     = castai_aks_cluster.this.id
  name           = "default"
  disk_cpu_ratio = 0
  min_disk_size  = 100
  subnets        = var.subnets

  aks {
    max_pods_per_node = 40
  }
}

resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_aks_cluster.this.id
  configuration_id = castai_node_configuration.default.id
}

resource "castai_node_template" "default_by_castai" {
  cluster_id = castai_aks_cluster.this.id

  name             = "default-by-castai"
  is_default       = true
  is_enabled       = true
  configuration_id = castai_node_configuration.default.id
  should_taint     = true

  constraints {
    on_demand = true
  }
}

resource "castai_node_template" "example_spot_template" {
  cluster_id = castai_aks_cluster.this.id

  name             = "example_spot_template"
  is_default       = false
  is_enabled       = true
  configuration_id = castai_node_configuration.default.id
  should_taint     = true

  custom_labels = {
    type = "spot"
  }

  custom_taints {
    key    = "dedicated"
    value  = "backend"
    effect = "NoSchedule"
  }

  constraints {
    spot                                        = true
    use_spot_fallbacks                          = true
    fallback_restore_rate_seconds               = 300
    enable_spot_diversity                       = true
    spot_diversity_price_increase_limit_percent = 20
    is_gpu_only                                 = false
    min_cpu                                     = 2
    max_cpu                                     = 8
    min_memory                                  = 4096
    max_memory                                  = 16384
    architectures                               = ["amd64"]
    burstable_instances                         = "disabled"
    customer_specific                           = "disabled"

    instance_families {
      exclude = ["standard_FSv2"]
    }

    custom_priority = [
      {
        instance_families = ["standard_Dv4"]
        spot              = true
      }
    ]
  }

}

resource "castai_autoscaler" "castai_autoscaler_policy" {
  cluster_id = castai_aks_cluster.this.id

  autoscaler_settings {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true
    }

    cluster_limits {
      enabled = false

      cpu {
        min_cores = 1
        max_cores = 200
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled = true
      }

      evictor {
        aggressive_mode           = false
        cycle_interval            = "60s"
        dry_run                   = false
        enabled                   = false
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }
  }
}
