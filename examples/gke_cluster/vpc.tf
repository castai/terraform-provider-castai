module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "5.0.0"
  project_id   = var.project_id
  network_name = "gke-network"
  subnets = [
    {
      subnet_name           = "ip-range-nodes"
      subnet_ip             = "10.0.0.0/16"
      subnet_region         = var.cluster_region
      subnet_private_access = "true"
    },
  ]

  secondary_ranges = {
    "ip-range-nodes" = [
      {
        range_name    = "ip-range-pods"
        ip_cidr_range =  "10.20.0.0/16"
      },
      {
        range_name    = "ip-range-services"
        ip_cidr_range =  "10.30.0.0/24"
      }
    ]
  }
}