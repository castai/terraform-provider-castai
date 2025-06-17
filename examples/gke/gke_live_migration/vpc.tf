# 1. Create VPC.

locals {
  ip_range_pods     = "${var.cluster_name}-ip-range-pods"
  ip_range_services = "${var.cluster_name}-ip-range-services"
  ip_range_nodes    = "${var.cluster_name}-ip-range-nodes"
}

module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "~> 10.0"
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
        ip_cidr_range = "10.20.0.0/16"
      },
      {
        range_name    = local.ip_range_services
        ip_cidr_range = "10.30.0.0/24"
      }
    ]
  }
}

resource "google_compute_firewall" "allow_ssh" {
  name    = "allow-ssh-${var.cluster_name}"
  network = var.cluster_name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  direction     = "INGRESS"
}
