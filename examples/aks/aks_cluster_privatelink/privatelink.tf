# This file holds everything specific to Private Link. For real deployments, this file is probably the only important bit.
#
# Two resources enable CAST AI private traffic:
#
# 1. A private endpoint that connects the VNet to the CAST AI Private Link Service via alias.
# 2. A private DNS zone that redirects CAST AI hostnames to the private endpoint NIC. This can be done per-host or for the whole zone.
# Approval of the private link connection happens automatically, as long as CAST AI is notified to include the given subscription in the allow list.


resource "azurerm_private_endpoint" "castai" {
  name                = "${local.name}-castai-pe"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  subnet_id           = azurerm_subnet.private_endpoints.id
  tags                = local.tags

  private_service_connection {
    name = "${local.name}-castai-pls"


    # The alias of the producer Private Link Service, see CAST AI docs for the possible values per region.
    # A PLS in one region can be used to access CAST AI deployment in another region as well.
    private_connection_resource_alias = var.private_link_service_alias

    # A connection made by alias is always a manual request.
    # If the subscription is in auto-approval list on producer side, this manual request gets approved immediately.
    is_manual_connection = true

    # Shows up in the producer's connection details to distinguish the request.
    request_message = "AKS Terraform example (examples/aks/aks_cluster_privatelink)"
  }
}

# ------------------------------------------------------------------------------
# DNS: the consumer's responsibility in Azure Private Link to route traffic correctly.
# This is just one example way to set this up - it can be a whole zone or list of hostnames.
# The important bit is that those hostnames end up going through the Private Link.
# ------------------------------------------------------------------------------

resource "azurerm_private_dns_zone" "castai" {
  name                = var.castai_private_dns_zone
  resource_group_name = azurerm_resource_group.this.name
  tags                = local.tags

  # Fail loudly instead of creating broken records when a URL variable points outside the zone.
  lifecycle {
    precondition {
      condition     = alltrue([for host in local.castai_hostnames : endswith(host, ".${var.castai_private_dns_zone}")])
      error_message = "Every castai_* URL hostname must live under the private DNS zone '${var.castai_private_dns_zone}'. Either fix the URL variables or ensure they fall within the private dns zone."
    }
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "castai" {
  name                 = "${local.name}-castai-dns-link"
  private_dns_zone_id  = azurerm_private_dns_zone.castai.id
  virtual_network_id   = azurerm_virtual_network.this.id
  registration_enabled = false
  tags                 = local.tags
}

# One record per CAST AI hostname the cluster talks to. Each name is relative to the zone above.
# For example "api" -> api.cast.ai, "kvisor.prod-master" -> "kvisor.prod-master.cast.ai"
#
# Caveat that comes with a parent zone like cast.ai: inside this VNet the zone is authoritative,
# so a hostname within the zone without record here, will not NOT fall back to the public DNS.
# This is useful for validation - "is this traffic private?" - but it might be undesired.
# authoritative, so a CAST AI hostname without a record here does NOT fall back
# to public DNS. That is what makes "is this traffic private?" unambiguous, but
# it also means every hostname CAST AI starts using has to be added here.
#
# Azure Private DNS has no wildcard records, so there is no "*.cast.ai" shortcut.
resource "azurerm_private_dns_a_record" "castai" {
  for_each = toset(local.castai_dns_records)

  name                = each.value
  private_dns_zone_id = azurerm_private_dns_zone.castai.id
  ttl                 = 60

  records = [azurerm_private_endpoint.castai.private_service_connection[0].private_ip_address]
  tags    = local.tags
}
