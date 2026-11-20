# Practice example for the `gke.local_ssd` block of `castai_node_configuration`
# (GKE local SSD storage recipe: EPHEMERAL / MIXED / BLOCK).

terraform {
  required_version = ">= 1.0"

  required_providers {
    castai = {
      source = "castai/castai"
    }
  }
}

# The provider picks CASTAI_API_URL / CASTAI_API_TOKEN up from the environment when
# the attributes are not set here, so a provider block is not required. For a local
# Tilt stack, point CASTAI_API_URL at https://api--<your-machine>.local.castai.co
# (see README.md for building this branch of the provider and using dev_overrides).

# provider "castai" {
#   api_url   = "https://api.cast.ai"
#   api_token = var.castai_api_token
# }

# Onboarding path: registers the GKE cluster with CAST AI. The agent itself (Helm
# install) is not covered by this example - for the full flow see the
# castai/gke-cluster/castai registry module (its node_configurations map does not
# support local_ssd yet).
resource "castai_gke_cluster" "onboarded" {
  count            = var.onboard_cluster ? 1 : 0
  name             = var.gke_cluster_name
  project_id       = var.gke_project_id
  location         = var.gke_cluster_location
  credentials_json = var.gke_credentials_json
}

locals {
  cluster_id = var.onboard_cluster ? castai_gke_cluster.onboarded[0].id : var.castai_cluster_id
}

resource "castai_node_configuration" "local_ssd" {
  name       = "local-ssd-demo"
  cluster_id = local.cluster_id
  subnets    = var.subnets

  disk_cpu_ratio = 0
  min_disk_size  = 100

  gke {
    local_ssd {
      mode                      = var.local_ssd_mode
      min_local_storage_gib     = var.min_local_storage_gib
      local_storage_per_cpu_gib = var.local_storage_per_cpu_gib
      ephemeral_storage_percent = var.ephemeral_storage_percent
      min_iops                  = var.min_iops
    }
  }
}

output "node_configuration_id" {
  description = "ID of the created node configuration."
  value       = castai_node_configuration.local_ssd.id
}
