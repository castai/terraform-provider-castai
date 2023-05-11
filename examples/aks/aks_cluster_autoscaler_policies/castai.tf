# 3. Connect AKS cluster to CAST AI in READ-ONLY mode.

# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

provider "castai" {
  api_url = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.this.kube_config.0.host
    client_certificate     = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_key)
    cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.cluster_ca_certificate)
  }
}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module.
module "castai-aks-cluster" {
  source = "castai/aks/castai"

  api_url = var.castai_api_url
  aks_cluster_name    = var.cluster_name
  aks_cluster_region  = var.cluster_region
  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio  = 25
      subnets         = [azurerm_subnet.internal.id]
      tags            = var.tags
    }

    test_node_config = {
      disk_cpu_ratio  = 25
      subnets         = [azurerm_subnet.internal.id]
      tags            = var.tags
      max_pods_per_node = 40
    }
  }

  node_templates = {
    spot_tmpl = {
      configuration_id = module.castai-aks-cluster.castai_node_configurations["default"]
      should_taint = true

      custom_labels = {
        custom-label-key-1 = "custom-label-value-1"
        custom-label-key-2 = "custom-label-value-2"
      }

      custom_taints = [
        {
          key = "custom-taint-key-1"
          value = "custom-taint-value-1"
        },
        {
          key = "custom-taint-key-2"
          value = "custom-taint-value-2"
        }
      ]

      constraints = {
        fallback_restore_rate_seconds = 1800
        spot = true
        use_spot_fallbacks = true
        min_cpu = 4
        max_cpu = 100
        instance_families = {
          exclude = ["standard_DPLSv5"]
        }
        compute_optimized = false
        storage_optimized = false
      }
    }
  }

  // Configure Autoscaler policies as per API specification https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies.
  // Here:
  //  - unschedulablePods - Unscheduled pods policy
  //  - spotInstances     - Spot instances configuration
  //  - nodeDownscaler    - Node deletion policy
  autoscaler_policies_json   = <<-EOT
    {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["azure"],
            "spotBackups": {
                "enabled": true
            },
            "spotDiversityEnabled": false
        },
        "nodeDownscaler": {
            "enabled": true,
            "emptyNodes": {
                "enabled": true
            },
            "evictor": {
                "aggressiveMode": false,
                "cycleInterval": "5m10s",
                "dryRun": false,
                "enabled": true,
                "nodeGracePeriodMinutes": 10,
                "scopedMode": false
            }
        },
        "clusterLimits": {
            "cpu": {
                "maxCores": 20,
                "minCores": 1
            },
            "enabled": true
        }
    }
  EOT

}
