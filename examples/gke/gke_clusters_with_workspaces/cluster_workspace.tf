data "castai_rebalancing_schedule" "data_rs" {
  count = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  name  = "org rebalancing schedule"
}

module "castai-gke-iam" {
  count                       = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  source                      = "castai/gke-iam/castai"
  version                     = "~> 0.5"
  project_id                  = var.gke_project_id
  gke_cluster_name            = var.gke_cluster_name
  service_accounts_unique_ids = length(var.service_accounts_unique_ids) == 0 ? [] : var.service_accounts_unique_ids
}

resource "castai_gke_cluster" "castai_cluster" {
  count                      = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  project_id                 = var.gke_project_id
  location                   = var.gke_cluster_location
  name                       = var.gke_cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  credentials_json           = terraform.workspace == var.org_workspace ? "" : module.castai-gke-iam[0].private_key
}


resource "castai_rebalancing_job" "foo-job" {
  count                   = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id              = castai_gke_cluster.castai_cluster[0].id
  rebalancing_schedule_id = data.castai_rebalancing_schedule.data_rs[0].id
  enabled                 = true
}

resource "castai_autoscaler" "castai_autoscaler_policy" {
  count      = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id = castai_gke_cluster.castai_cluster[0].id

  autoscaler_settings {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true
    }

    cluster_limits {
      enabled = false

      cpu {
        min_cores = 1
        max_cores = 200
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled = true
      }

      evictor {
        aggressive_mode           = false
        cycle_interval            = "60s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }
  }
}

resource "castai_node_configuration" "default" {
  count          = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id     = castai_gke_cluster.castai_cluster[0].id
  name           = "default"
  disk_cpu_ratio = 0
  min_disk_size  = 100
  subnets        = var.gke_subnets
}

resource "castai_node_configuration_default" "this" {
  count            = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id       = castai_gke_cluster.castai_cluster[0].id
  configuration_id = castai_node_configuration.default[0].id
}

resource "castai_node_template" "default_by_castai" {
  count      = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id = castai_gke_cluster.castai_cluster[0].id

  name             = "default-by-castai"
  is_default       = true
  is_enabled       = true
  configuration_id = castai_node_configuration.default[0].id
  should_taint     = false

  constraints {
    on_demand = true
  }
}

resource "castai_node_template" "example_spot_template" {
  count      = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  cluster_id = castai_gke_cluster.castai_cluster[0].id

  name                     = "example_spot_template"
  is_default               = false
  is_enabled               = true
  configuration_id         = castai_node_configuration.default[0].id
  should_taint             = true
  custom_instances_enabled = true # gke specific

  custom_labels = {
    type = "spot"
  }

  custom_taints {
    key    = "dedicated"
    value  = "backend"
    effect = "NoSchedule"
  }

  constraints {
    spot                                        = true
    use_spot_fallbacks                          = true
    fallback_restore_rate_seconds               = 1800
    enable_spot_diversity                       = true
    spot_diversity_price_increase_limit_percent = 20
    min_cpu                                     = 2
    max_cpu                                     = 8
    min_memory                                  = 4096
    max_memory                                  = 16384
    architectures                               = ["amd64"]
    burstable_instances                         = "disabled"
    customer_specific                           = "enabled"

    instance_families {
      exclude = ["e2"]
    }
    custom_priority {
      instance_families = ["c4"]
      spot              = true
    }
  }
}
