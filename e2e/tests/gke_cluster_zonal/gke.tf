module "gke" {
  source                     = "terraform-google-modules/kubernetes-engine/google"
  version                    = "20.0.0"
  regional                   = false
  project_id                 = var.project_id
  name                       = var.cluster_name
  zones                      = var.cluster_zones
  network                    = module.vpc.network_name
  subnetwork                 = module.vpc.subnets_names[0]
  ip_range_pods              = local.ip_range_pods
  ip_range_services          = local.ip_range_services
  http_load_balancing        = false
  network_policy             = false
  horizontal_pod_autoscaling = true
  filestore_csi_driver       = false

  node_pools = [
    {
      name               = "default-node-pool"
      machine_type       = "e2-medium"
      node_locations     = var.cluster_zones[0]
      min_count          = 1
      max_count          = 100
      local_ssd_count    = 0
      disk_size_gb       = 100
      disk_type          = "pd-standard"
      image_type         = "COS_CONTAINERD"
      auto_repair        = true
      auto_upgrade       = true
      preemptible        = false
      initial_node_count = 1
    },
  ]
}