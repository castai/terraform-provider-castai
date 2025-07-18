# 3. Connect GKE cluster to CAST AI in read-only mode.

# Configure Data sources and providers required for CAST AI connection.

data "google_client_config" "default" {}

data "google_container_cluster" "my_cluster" {
  name     = var.cluster_name
  location = var.cluster_region
  project  = var.project_id
}

# Configure GKE cluster connection using CAST AI gke-cluster module.
module "castai-gke-iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.5"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}

module "castai-gke-cluster" {
  source  = "castai/gke-cluster/castai"
  version = "~> 8.0"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = var.cluster_region

  gke_credentials            = module.castai-gke-iam.private_key
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration  = module.castai-gke-cluster.castai_node_configurations["default"]
  install_workload_autoscaler = true

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
      name             = "default-by-castai"
      configuration_id = module.castai-gke-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand = true
      }
    }

    example_spot_template = {
      configuration_id         = module.castai-gke-cluster.castai_node_configurations["default"]
      is_enabled               = true
      should_taint             = true
      custom_instances_enabled = false # custom_instances_enabled should be set to same value(true or false) at Node templates & unschedulable_pods policy for backward compatability

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
          exclude = ["e2"]
        }
        custom_priority = {
          instance_families = ["c5"]
          spot              = true
        }
      }
    }
  }

  autoscaler_settings = {
    enabled                                 = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled                  = false
      custom_instances_enabled = false # custom_instances_enabled should be set to same value(true or false) at Node templates & unschedulable_pods policy for backward compatability
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

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-gke-cluster" has to be destroyed before module "castai-gke-iam" and "module.gke"
  depends_on = [data.google_container_cluster.my_cluster, module.castai-gke-iam]
}
