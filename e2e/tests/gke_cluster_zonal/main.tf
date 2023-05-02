data "google_client_config" "default" {}


provider "helm" {
  kubernetes {
    host                   = "https://${module.gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
}

provider "castai" {
  api_token = var.castai_api_token
  api_url  = var.castai_api_url
}

provider "google" {
  credentials = base64decode(var.gcp_credentials_base64)
  region = var.network_region
}

provider "google-beta" {
  credentials = base64decode(var.gcp_credentials_base64)
  region = var.network_region
}

module "castai-gke-iam" {
  source = "castai/gke-iam/castai"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name

}

module "castai-gke-cluster" {
  source = "castai/gke-cluster/castai"

  project_id         = var.project_id
  gke_cluster_name   = var.cluster_name
  gke_cluster_location = var.cluster_location
  api_url = var.castai_api_url
  gke_credentials            = module.castai-gke-iam.private_key
  delete_nodes_on_disconnect = true

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [module.vpc.subnets_ids[0]]
    }

    test_node_config = {
      disk_cpu_ratio    = 10
      subnets           = [module.vpc.subnets_ids[0]]
      max_pods_per_node = 40
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
          key   = "custom-taint-key-1"
          value = "custom-taint-value-1"
        },
        {
          key    = "custom-taint-key-2"
          value  = "custom-taint-value-2"
          effect = "NoSchedule"
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

  # Full schema can be found here https://api.cast.ai/v1/spec/#/PoliciesAPI/PoliciesAPIUpsertClusterPolicies
  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "isScopedMode": true,
        "unschedulablePods": {
            "enabled": true
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["gcp"],
            "spotBackups": {
                "enabled": true
            }
        },
        "nodeDownscaler": {
            "emptyNodes": {
                "enabled": true
            }
        }
    }
  EOT

  depends_on = [module.gke, module.castai-gke-iam]
}
