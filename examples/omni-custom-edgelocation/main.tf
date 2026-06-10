provider "castai" {
  api_token = var.castai_api_token
}

resource "castai_edge_location" "this" {
  name               = var.edge_location_name
  cluster_id         = var.cluster_id
  organization_id    = var.organization_id
  description        = var.description
  region             = var.region
  control_plane_mode = "SHARED"

  custom = {}

  networking = {
    tunneled_cidrs = var.tunneled_cidrs

    cni = {
      overlay       = var.cni_overlay
      overlay_encap = var.cni_overlay_encap
    }
  }
}

