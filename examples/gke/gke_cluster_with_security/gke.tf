# 2. Create GKE cluster.

data "google_client_config" "default" {}

resource "google_container_cluster" "my-k8s-cluster" {

  initial_node_count = "3"
  node_config {
    machine_type = "n2-standard-2" # default nodes - not enough mem for cast agent
    preemptible  = false
  }

  location = var.cluster_region

  project = var.project_id

  name       = var.cluster_name
  network    = module.vpc.network_name
  subnetwork = module.vpc.subnets_names[0]

  enable_autopilot         = "false"
  enable_kubernetes_alpha  = "false"
  enable_l4_ilb_subsetting = "false"
  enable_legacy_abac       = "false"
  enable_tpu               = "false"

  node_pool_defaults {
    node_config_defaults {
      logging_variant = "DEFAULT"
    }
  }

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_secondary_range_name  = local.ip_range_pods     # Must match the range_name in subnet
    services_secondary_range_name = local.ip_range_services # Must match the range_name in subnet
    stack_type                    = "IPV4"
  }
}
