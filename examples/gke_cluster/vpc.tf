module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "5.0.0"
  project_id   = var.project_id
  network_name = var.network_name
  subnets = [
    {
      subnet_name           = var.ip_range_nodes_name
      subnet_ip             = var.ip_range_nodes_cidr
      subnet_region         = var.cluster_region
      subnet_private_access = "true"
    },
  ]

  secondary_ranges = {
    (var.ip_range_nodes_name) = [
      {
        range_name    = var.ip_range_pods_name
        ip_cidr_range = var.ip_range_pods_cidr
      },
      {
        range_name    = var.ip_range_services_name
        ip_cidr_range = var.ip_range_services_cidr
      }
    ]
  }
}