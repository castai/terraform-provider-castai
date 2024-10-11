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

resource "castai_gke_cluster_id" "this" {
  project_id = var.project_id
  name       = var.cluster_name
  location   = module.gke.location
}

resource "helm_release" "castai_cloud_proxy" {
  name             = "castai-cloud-proxy"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-cloud-proxy"
  version          = var.cloud_proxy_version
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  values = var.cloud_proxy_values

  set {
    name  = "castai.clusterID"
    value = castai_gke_cluster_id.this.id
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_gke_cluster_id.this.cluster_token
  }

  set {
    name  = "castai.grpcURL"
    value = var.cloud_proxy_grpc_url
  }

  depends_on = [module.gke, module.castai-gke-iam]
}

module "castai-gke-cluster" {
  count = var.cluster_read_only ? 0 : 1

  source = "castai/gke-cluster/castai"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  wait_for_cluster_ready = true

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = module.gke.location

  gke_credentials                 = var.cluster_read_only ? null : "{}"
  default_node_configuration_name = "default"

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

  depends_on = [module.gke, module.castai-gke-iam, helm_release.castai_cloud_proxy]
}

