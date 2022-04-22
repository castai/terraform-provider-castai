module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "5.0.0"
  project_id   = var.project_id
  network_name = var.cluster_name
  subnets = [
    {
      subnet_name           = local.ip_range_nodes
      subnet_ip             = "10.0.0.0/16"
      subnet_region         = var.cluster_region
      subnet_private_access = "true"
    },
  ]

  secondary_ranges = {
    (local.ip_range_nodes) = [
      {
        range_name    = local.ip_range_pods
        ip_cidr_range =  "10.1.0.0/20"
      },
      {
        range_name    = local.ip_range_services
        ip_cidr_range =  "10.3.0.0/20"
      }
    ]
  }
}
