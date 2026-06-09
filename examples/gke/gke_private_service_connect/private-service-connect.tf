# 2. PSC Endpoint setup

resource "google_compute_address" "cast_ai_private_api" {
  project      = var.project_id
  name         = "cast-ai-private-api"
  region       = module.vpc.subnets_regions[1]
  address_type = "INTERNAL"
  subnetwork   = module.vpc.subnets_self_links[1]
  address      = cidrhost(module.vpc.subnets_ips[1], 2)
}

resource "google_compute_forwarding_rule" "cast_ai_private_api" {
  project                 = var.project_id
  name                    = "cast-ai-private-api"
  target                  = var.cast_api_service_attachment_uri
  network                 = module.vpc.network_id
  region                  = module.vpc.subnets_regions[1]
  ip_address              = google_compute_address.cast_ai_private_api.id
  load_balancing_scheme   = ""
  allow_psc_global_access = var.allow_psc_global_access
}


# 3. DNS setup

resource "google_dns_managed_zone" "psc_zone" {
  name        = "cast-ai-psc-zone"
  project     = var.project_id
  dns_name    = "${var.castai_api_private_domain}."
  description = "Cast AI Private Service Connect zone"

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = module.vpc.network_id
    }
  }
}

resource "google_dns_record_set" "a" {
  name         = "*.psc.${google_dns_managed_zone.psc_zone.dns_name}"
  project      = var.project_id
  managed_zone = google_dns_managed_zone.psc_zone.name
  type         = "A"
  ttl          = 300

  rrdatas = [google_compute_address.cast_ai_private_api.address]
}