# 3. Connect GKE cluster to CAST AI in read-only mode.

# Configure Data sources and providers required for CAST AI connection.

data "google_client_config" "default" {}

data "google_secret_manager_secret_version" "cast_ai_services_dev_token" {
  secret  = "rouseservice-key"
  project = var.project_id
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = data.google_secret_manager_secret_version.cast_ai_services_dev_token.secret_data
}

data "google_container_cluster" "my_cluster" {
  name     = var.cluster_name
  location = var.cluster_region
  project  = var.project_id
}



provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.my_cluster.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.my_cluster.master_auth.0.cluster_ca_certificate)
  }
}

# Configure GKE cluster connection using CAST AI gke-cluster module.
module "castai-gke-iam" {
  source = "castai/gke-iam/castai"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}

module "castai-gke-cluster" {
  source = "castai/gke-cluster/castai"

  api_url                = var.castai_api_url
  castai_api_token       = data.google_secret_manager_secret_version.cast_ai_services_dev_token.secret_data
  wait_for_cluster_ready = true

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = var.cluster_region

  gke_credentials            = data.google_container_cluster.my_cluster.master_auth[0].client_certificate
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-gke-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = var.subnets
      tags           = var.tags
    }

    # # Commented out for POC
    # test_node_config = {
    #   disk_cpu_ratio    = 10
    #   subnets           = [module.vpc.subnets_ids[0]]
    #   tags              = var.tags
    #   max_pods_per_node = 40
    #   disk_type         = "pd-ssd",
    #   network_tags      = ["dev"]
    # }

  }
  # # Commented out for POC
  #   node_templates = {
  #     default_by_castai = {
  #       name = "default-by-castai"
  #       configuration_id = module.castai-gke-cluster.castai_node_configurations["default"]
  #       is_default   = true
  #       should_taint = false

  #       constraints = {
  #         on_demand          = true
  #         spot               = true
  #         use_spot_fallbacks = true

  #         enable_spot_diversity                       = false
  #         spot_diversity_price_increase_limit_percent = 20
  #       }
  #     }
  #     spot_tmpl = {
  #       configuration_id = module.castai-gke-cluster.castai_node_configurations["default"]
  #       should_taint     = true

  #       custom_labels = {
  #         custom-label-key-1 = "custom-label-value-1"
  #         custom-label-key-2 = "custom-label-value-2"
  #       }

  #       custom_taints = [
  #         {
  #           key = "custom-taint-key-1"
  #           value = "custom-taint-value-1"
  #           effect = "NoSchedule"
  #         },
  #         {
  #           key = "custom-taint-key-2"
  #           value = "custom-taint-value-2"
  #           effect = "NoSchedule"
  #         }
  #       ]

  #       constraints = {
  #         fallback_restore_rate_seconds = 1800
  #         spot                          = true
  #         use_spot_fallbacks            = true
  #         min_cpu                       = 4
  #         max_cpu                       = 100
  #         instance_families             = {
  #           exclude = ["e2"]
  #         }
  #         compute_optimized = false
  #         storage_optimized = false
  #       }

  #       custom_instances_enabled = true
  #     }
  #   }

  // Configure Autoscaler policies as per API specification https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies.
  // Here:
  //  - unschedulablePods - Unscheduled pods policy
  //  - nodeDownscaler    - Node deletion policy

  # # Commend oout for POC
  autoscaler_policies_json = <<-EOT
{
  "enabled": false,
  "unschedulablePods": {
    "enabled": false
  },
  "nodeDownscaler": {
    "enabled": false,
    "emptyNodes": {
      "enabled": false
    },
    "evictor": {
      "aggressiveMode": false,
      "cycleInterval": "5m10s",
      "dryRun": false,
      "enabled": false,
      "nodeGracePeriodMinutes": 10,
      "scopedMode": false
    }
  },
  "clusterLimits": {
    "cpu": {
      "maxCores": 20,
      "minCores": 1
    },
    "enabled": false
  },
  "maxReclaimRate": 0,
  "spotBackups": {
    "enabled": false,
    "spotBackupRestoreRateSeconds": 1800
  }
}
  EOT

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-gke-cluster" has to be destroyed before module "castai-gke-iam" and "module.gke"
  depends_on = [data.google_container_cluster.my_cluster, module.castai-gke-iam]
}
