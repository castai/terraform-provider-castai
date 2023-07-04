# 3. Connect GKE cluster to CAST AI in read-only mode.

# Configure Data sources and providers required for CAST AI connection.

data "google_client_config" "default" {}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = "https://${module.gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
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

  api_url = var.castai_api_url

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = module.gke.location

  gke_credentials            = module.castai-gke-iam.private_key
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-gke-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [module.vpc.subnets_ids[0]]
      tags           = var.tags
    }

    test_node_config = {
      disk_cpu_ratio    = 10
      subnets           = [module.vpc.subnets_ids[0]]
      tags              = var.tags
      max_pods_per_node = 40
      disk_type         = "pd-ssd",
      network_tags      = ["dev"]
    }

  }

  node_templates = {
    spot_tmpl = {
      configuration_id = module.castai-gke-cluster.castai_node_configurations["default"]
      should_taint     = true

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
        spot                          = true
        use_spot_fallbacks            = true
        min_cpu                       = 4
        max_cpu                       = 100
        instance_families             = {
          exclude = ["e2"]
        }
        compute_optimized = false
        storage_optimized = false
      }

      custom_instances_enabled = true
    }
  }

  // Configure Autoscaler policies as per API specification https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies.
  // Here:
  //  - unschedulablePods - Unscheduled pods policy
  //  - spotInstances     - Spot instances configuration
  //  - nodeDownscaler    - Node deletion policy
  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["gcp"],
            "spotBackups": {
                "enabled": true
            },
            "spotDiversityEnabled": false,
            "spotDiversityPriceIncreaseLimitPercent": 20
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

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-gke-cluster" has to be destroyed before module "castai-gke-iam" and "module.gke"
  depends_on = [module.gke, module.castai-gke-iam]
}
