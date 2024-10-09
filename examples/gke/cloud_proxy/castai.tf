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

  create_service_account              = false
  setup_cloud_proxy_workload_identity = true
}

module "castai-gke-cluster" {
  source = "castai/gke-cluster/castai"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = module.gke.location

  gke_credentials                 = var.cluster_read_only ? null : "{}"
  default_node_configuration_name = "default"

  install_cloud_proxy = true

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [module.vpc.subnets_ids[0]]
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
  }

  depends_on = [module.gke, module.castai-gke-iam]
}

