# 3. Connect GKE cluster to CAST AI with WOOP (workload autoscaler) and evictor (V2 API).

# Configure Data sources and providers required for CAST AI connection.

locals {
  init_script = var.gke_img_type == "COS_CONTAINERD" ? "init_cos.sh" : "init_ubuntu.sh"
}

data "google_client_config" "default" {}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes = {
    host                   = "https://${module.gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
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
  version = "~> 10.1"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true
  project_id             = var.project_id
  gke_cluster_name       = var.cluster_name
  gke_cluster_location   = module.gke.location

  gke_credentials            = module.castai-gke-iam.private_key
  delete_nodes_on_disconnect = true

  default_node_configuration_name = "default"

  # WOOP (workload autoscaler) installs the workload autoscaler component so
  # CAST AI can generate scheduling recommendations based on workload needs.
  install_workload_autoscaler = true

  # CLM (Continuous Live Migration) installs the live migration component so
  # CAST AI can migrate workloads off nodes being drained.
  install_live = true

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [module.vpc.subnets_ids[0]]
      # https://cloud.google.com/container-optimized-os/docs/release-notes/m121
      image       = "projects/cos-cloud/global/images/cos-121-18867-90-59"
      init_script = base64encode(file(local.init_script))
    }
  }

  node_templates = {
    default_by_castai = {
      name               = "default-by-castai"
      configuration_name = "default"
      is_default         = true
      is_enabled         = true
      should_taint       = false
      clm_enabled        = true

      constraints = {
        on_demand          = true
        spot               = true
        use_spot_fallbacks = true

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20
      }
    }
  }

  autoscaler_settings = {
    enabled                                 = true
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true

      pod_pinner = {
        enabled = true
      }
    }

    node_downscaler = {
      enabled = true

      empty_nodes = {
        enabled = true
      }

      # No evictor block here: evictor is managed via the V2 castai_evictor
      # resource below to avoid dual-write conflicts with the v1 settings.
    }

    cluster_limits = {
      enabled = true

      cpu = {
        max_cores = 20
        min_cores = 1
      }
    }
  }

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-gke-cluster" has to be destroyed before module "castai-gke-iam" and "module.gke"
  depends_on = [module.gke, module.castai-gke-iam]
}

# ── Evictor config via V2 API ──
# Uses castai_evictor (V2) instead of autoscaler_settings.node_downscaler.evictor (v1)
# to avoid dual-write conflicts. The CAST AI control plane syncs this config to the
# EvictorConfig CRD on the next snapshot.

resource "castai_evictor" "this" {
  cluster_id = module.castai-gke-cluster.cluster_id

  enabled         = true
  aggressive_mode = true

  depends_on = [module.castai-gke-cluster]
}

# ── WOOP workload scaling policy ──
# Generic policy applied to all workloads for CPU and memory right-sizing.

resource "castai_workload_scaling_policy" "default" {
  cluster_id = module.castai-gke-cluster.cluster_id

  name              = "default"
  apply_type        = "DEFERRED"
  management_option = "MANAGED"

  cpu {
    function = "QUANTILE"
    overhead = 0.1
    apply_threshold_strategy {
      type       = "PERCENTAGE"
      percentage = 0.1
    }
    args                     = ["0.9"]
    look_back_period_seconds = 172800
    min                      = 0.1
    max                      = 10
  }

  memory {
    function = "MAX"
    overhead = 0.15
    apply_threshold_strategy {
      type = "DEFAULT_ADAPTIVE"
    }
    limit {
      type       = "MULTIPLIER"
      multiplier = 1.5
    }
  }

  confidence {
    threshold = 0.9
  }

  depends_on = [module.castai-gke-cluster]
}

# ── Rebalancing schedule ──
# Runs every 10 minutes, only triggers if 5% savings is achieved.

resource "castai_rebalancing_schedule" "this" {
  name = "woop-evictor-rebalancing"
  schedule {
    cron = "*/10 * * * *"
  }
  trigger_conditions {
    savings_percentage = 5
  }
  launch_configuration {
    rebalancing_min_nodes = 2
    execution_conditions {
      achieved_savings_percentage = 5
      enabled                     = true
    }
  }
}

resource "castai_rebalancing_job" "this" {
  cluster_id              = module.castai-gke-cluster.cluster_id
  rebalancing_schedule_id = castai_rebalancing_schedule.this.id
  enabled                 = true

  depends_on = [module.castai-gke-cluster]
}

# ── Outputs ──

output "cluster_id" {
  value = module.castai-gke-cluster.cluster_id
}

output "evictor_status" {
  value       = castai_evictor.this.status
  description = "Should show 'Compatible' if the evictor chart version supports config sync"
}
