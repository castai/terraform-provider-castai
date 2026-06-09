provider "castai" {
  api_token = var.castai_api_token
}

resource "castai_edge_location" "this1" {
  name               = "customedgelocation01"
  cluster_id         = var.cluster_id
  organization_id    = var.organization_id
  description        = var.description
  region             = var.region
  control_plane_mode = "SHARED"

  custom = {}

  networking = {
    tunneled_cidrs = var.tunneled_cidrs

    cni = {
      overlay       = "OVERLAY_FULL"
      overlay_encap = "OVERLAY_ENCAP_FOU"
    }
  }
}

resource "castai_edge_location" "this2" {
  name               = "customedgelocation02"
  cluster_id         = var.cluster_id
  organization_id    = var.organization_id
  description        = var.description
  region             = var.region
  control_plane_mode = "SHARED"

  custom = {}

  networking = {
    tunneled_cidrs = ["10.0.0.0/8"]
    cni = {
      overlay = "OVERLAY_OFF"
    }
  }
}
