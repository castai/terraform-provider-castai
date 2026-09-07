# 3. Connect GKE cluster to CAST AI with WOOP (workload autoscaler) and evictor (V2 API).
#
# This example uses the CAST AI umbrella Helm chart (castai-helm/castai) with
# tags.full=true instead of the castai/gke-cluster module for Helm releases.
# Terraform manages the cluster registration, GCP IAM, node configurations,
# autoscaler settings, evictor (V2), WOOP policy, and rebalancing schedule.
# CAST AI does NOT auto-upgrade the umbrella chart, so there is no version
# conflict with the Terraform-managed helm_release.

# Configure data sources and providers required for CAST AI connection.

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

# GCP IAM service account and permissions required for CAST AI node provisioning.
module "castai-gke-iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.5"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}

# Register the GKE cluster with CAST AI. This replaces the castai/gke-cluster
# module — it is a direct resource that returns cluster_id and cluster_token.
resource "castai_gke_cluster" "this" {
  project_id                 = var.project_id
  location                   = module.gke.location
  name                       = var.cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  credentials_json = module.castai-gke-iam.private_key

  depends_on = [module.gke, module.castai-gke-iam]
}

# Install the CAST AI umbrella Helm chart with tags.full=true.
# This deploys agent, cluster-controller, evictor, pod-mutator, pod-pinner,
# live, workload-autoscaler, spot-handler, kvisor, and
# workload-autoscaler-exporter in a single release.
resource "helm_release" "castai_umbrella" {
  name             = "castai"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai"
  version          = "0.43.170"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true

  set = [
    {
      name  = "global.castai.provider"
      value = "gke"
    },
    {
      name  = "global.castai.apiURL"
      value = var.castai_api_url
    },
    {
      name  = "global.castai.grpcURL"
      value = var.castai_grpc_url
    },
    {
      name  = "tags.full"
      value = "true"
    },
  ]

  set_sensitive = [
    {
      name  = "global.castai.apiKey"
      value = castai_gke_cluster.this.cluster_token
    },
  ]

  depends_on = [module.gke, castai_gke_cluster.this]
}

# ── Node configuration ──
# Default node configuration: disk/cpu ratio, subnets, image, and the CLM
# init script (cri-proxy) required for Live Migration.

resource "castai_node_configuration" "default" {
  cluster_id     = castai_gke_cluster.this.id
  name           = "default"
  disk_cpu_ratio = 25
  subnets        = [module.vpc.subnets_ids[0]]
  # https://cloud.google.com/container-optimized-os/docs/release-notes/m121
  image       = "projects/cos-cloud/global/images/cos-121-18867-90-59"
  init_script = base64encode(file(local.init_script))

  depends_on = [castai_gke_cluster.this]
}

# Promote the node configuration as the default.
resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_gke_cluster.this.id
  configuration_id = castai_node_configuration.default.id

  depends_on = [castai_node_configuration.default]
}

# ── Node template ──
# Default template with CLM (Continuous Live Migration) enabled.

resource "castai_node_template" "default_by_castai" {
  cluster_id = castai_gke_cluster.this.id

  name             = "default-by-castai"
  is_default       = true
  is_enabled       = true
  configuration_id = castai_node_configuration.default.id
  should_taint     = false
  clm_enabled      = true

  constraints {
    on_demand                                   = true
    spot                                        = true
    use_spot_fallbacks                          = true
    enable_spot_diversity                       = false
    spot_diversity_price_increase_limit_percent = 20
  }

  depends_on = [castai_node_configuration.default]
}

# ── Autoscaler settings ──
# No evictor block here: evictor is managed via the V2 castai_evictor
# resource below to avoid dual-write conflicts with the v1 settings.

resource "castai_autoscaler" "this" {
  cluster_id = castai_gke_cluster.this.id

  autoscaler_settings {
    enabled                                 = true
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true

      pod_pinner {
        enabled = true
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled = true
      }
    }

    cluster_limits {
      enabled = true

      cpu {
        max_cores = 20
        min_cores = 1
      }
    }
  }

  depends_on = [castai_gke_cluster.this, castai_node_template.default_by_castai]
}

# ── Evictor config via V2 API ──
# Uses castai_evictor (V2) instead of autoscaler_settings.node_downscaler.evictor (v1)
# to avoid dual-write conflicts. The CAST AI control plane syncs this config to the
# EvictorConfig CRD on the next snapshot.

resource "castai_evictor" "this" {
  cluster_id = castai_gke_cluster.this.id

  enabled         = true
  aggressive_mode = true

  depends_on = [castai_gke_cluster.this]
}

# ── WOOP workload scaling policy ──
# Generic policy applied to all workloads for CPU and memory right-sizing.

resource "castai_workload_scaling_policy" "default" {
  cluster_id = castai_gke_cluster.this.id

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
    constraints {
      min {
        constant = 0.1
      }
      max {
        constant = 10
      }
    }
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

  depends_on = [castai_gke_cluster.this]
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
  cluster_id              = castai_gke_cluster.this.id
  rebalancing_schedule_id = castai_rebalancing_schedule.this.id
  enabled                 = true

  depends_on = [castai_gke_cluster.this]
}

# ── Outputs ──

output "cluster_id" {
  value       = castai_gke_cluster.this.id
  description = "CAST AI cluster ID."
}

output "cluster_token" {
  value       = castai_gke_cluster.this.cluster_token
  description = "CAST AI cluster token used by the umbrella chart to authenticate to CAST AI."
  sensitive   = true
}

output "evictor_status" {
  value       = castai_evictor.this.status
  description = "Should show 'Compatible' if the evictor chart version supports config sync"
}
