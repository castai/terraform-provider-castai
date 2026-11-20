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

  # CAST AI hostnames. Validated against the zone resource that they live under its private DNS zone.
  # Each URL is expected to be "https://host" or "host:443", only <host> matters here.
  castai_urls = [
    var.castai_api_url,
    var.castai_grpc_url,
    var.castai_api_grpc_url,
    var.castai_files_url,
    var.castai_kvisor_grpc_addr,
    var.castai_telemetry_url,
  ]
  castai_hostnames = [for url in local.castai_urls : regex("^(?:[a-zA-Z][a-zA-Z0-9+.-]*://)?([^:/]+)", url)[0]]

  tags = merge(
    {
      environment = "example"
      managed_by  = "terraform"
      module      = "examples/aks/aks_cluster_privatelink"
    },
    var.tags,
  )
}
