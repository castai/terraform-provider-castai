locals {
  name = var.cluster_name

  # Address plan. Deliberately distinct from anything in the example readme so
  # overlapping with a real network is obvious.
  #
  #   10.20.0.0/16   VNet
  #   10.20.0.0/20   aks-nodes           node IPs and LB frontends only -
  #                                      Azure CNI Overlay keeps pod IPs out of
  #                                      the VNet
  #   10.20.16.0/24  private-endpoints   private endpoint NIC
  #   10.21.0.0/16   service_cidr        must not overlap the VNet
  #   192.168.0.0/16 pod_cidr            overlay-only; Azure reserves
  #                                      169.254/16, 172.30/16, 172.31/16 and
  #                                      192.0.2/24 - none of them used here
  vnet_address_space = "10.20.0.0/16"
  aks_subnet_prefix  = "10.20.0.0/20"
  pe_subnet_prefix   = "10.20.16.0/24"

  service_cidr   = "10.21.0.0/16"
  dns_service_ip = "10.21.0.10"
  pod_cidr       = "192.168.0.0/16"

  # CAST AI hostnames, parsed from the endpoint URL variables - the single
  # source of truth for both the module inputs and the private DNS records.
  # Each URL is "https://host" or "host:443"; only the host part matters here.
  castai_urls = [
    var.castai_api_url,
    var.castai_grpc_url,
    var.castai_api_grpc_url,
    var.castai_files_url,
    var.castai_kvisor_grpc_addr,
    var.castai_telemetry_url,
  ]
  castai_hostnames = [for url in local.castai_urls : regex("^(?:[a-zA-Z][a-zA-Z0-9+.-]*://)?([^:/]+)", url)[0]]

  # Record names relative to the private DNS zone: "api.cast.ai" -> "api",
  # "kvisor.prod-eu.cast.ai" -> "kvisor.prod-eu". The zone resource carries a
  # precondition that every hostname actually lives under the zone.
  castai_dns_records = distinct(concat(
    [for host in local.castai_hostnames : trimsuffix(host, ".${var.castai_private_dns_zone}")],
    var.castai_extra_dns_hostnames,
  ))

  tags = merge(
    {
      environment = "example"
      managed_by  = "terraform"
      module      = "examples/aks/aks_cluster_privatelink"
    },
    var.tags,
  )
}
