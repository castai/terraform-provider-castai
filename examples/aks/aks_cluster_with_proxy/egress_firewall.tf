
resource "azurerm_firewall" "egress_firewall" {
  name                = "aks-firewall"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  sku_name            = "AZFW_VNet"
  sku_tier            = "Standard"
  dns_proxy_enabled   = true

  ip_configuration {
    name                 = "configuration"
    subnet_id            = azurerm_subnet.egress_firewall_subnet.id
    public_ip_address_id = azurerm_public_ip.firewall_public_ip.id
  }

  lifecycle {
    ignore_changes = [
      tags["CreatedAt"]
    ]
  }
}

resource "azurerm_public_ip" "firewall_public_ip" {
  name                = "firewall-public-ip"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  allocation_method   = "Static"
  sku                 = "Standard"
}


resource "azurerm_firewall_network_rule_collection" "egress_rule" {
  name                = "egress-rule"
  azure_firewall_name = azurerm_firewall.egress_firewall.name
  resource_group_name = azurerm_resource_group.rg.name
  priority            = 100
  action              = "Allow"

  rule {
    name                  = "apiudp"
    protocols             = ["UDP"]
    source_addresses      = ["10.42.1.0/24"]
    destination_addresses = ["AzureCloud.${azurerm_resource_group.rg.location}"]
    destination_ports     = ["1194"]
  }
  rule {
    name                  = "apitcp"
    protocols             = ["TCP"]
    source_addresses      = ["10.42.1.0/24"]
    destination_addresses = ["AzureCloud.${azurerm_resource_group.rg.location}"]
    destination_ports     = ["9000"]
  }
  rule {
    name                  = "time"
    protocols             = ["UDP"]
    source_addresses      = ["10.42.1.0/24"]
    destination_addresses = ["*"]
    destination_ports     = ["123"]
  }
  rule {
    name              = "ghcr"
    protocols         = ["TCP"]
    source_addresses  = ["10.42.1.0/24"]
    destination_fqdns = ["ghcr.io", "pkg-containers.githubusercontent.com"]
    destination_ports = ["443"]
  }
  rule {
    name             = "docker"
    protocols        = ["TCP"]
    source_addresses = ["10.42.1.0/24"]
    destination_fqdns = [
      # Some registries used by system images or CAST AI images.
      "docker.io",
      "registry-1.docker.io",
      "production.cloudflare.docker.com",
      "registry.k8s.io",
      "us-docker.pkg.dev",
      "europe-west2-docker.pkg.dev",
      "prod-registry-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com"
    ]
    destination_ports = ["443"]
  }
  rule {
    name                  = "dns"
    source_addresses      = ["10.42.1.0/24"]
    destination_ports     = ["53"]
    destination_addresses = ["*"]
    protocols = [
      "Any"
    ]
  }

  dynamic "rule" {
    for_each = length(var.fqdn_without_proxy) > 0 ? [1] : []
    content {
      name              = "allowed_fqdns"
      protocols         = ["Any"]
      source_addresses  = ["10.42.1.0/24"]
      destination_fqdns = var.fqdn_without_proxy
      destination_ports = ["80", "443"]
    }
  }
}


resource "azurerm_firewall_network_rule_collection" "servicetags" {
  name                = "servicetags"
  azure_firewall_name = azurerm_firewall.egress_firewall.name
  resource_group_name = azurerm_resource_group.rg.name
  priority            = 200
  action              = "Allow"

  rule {
    name              = "allow service tags"
    source_addresses  = ["10.42.1.0/24"]
    destination_ports = ["*"]
    destination_addresses = [
      "AzureContainerRegistry.${azurerm_resource_group.rg.location}",
      "MicrosoftContainerRegistry.${azurerm_resource_group.rg.location}",
      "AzureActiveDirectory",
      "AzureMonitor",
      "AzureWebPubSub",
      "Storage",
      "StorageSyncService"
    ]
    protocols = [
      "Any"
    ]
  }
}

resource "azurerm_firewall_application_rule_collection" "fw-aks-google" {
  name                = "forbid-gcs-explicit"
  azure_firewall_name = azurerm_firewall.egress_firewall.name
  resource_group_name = azurerm_resource_group.rg.name
  priority            = 300
  action              = "Deny"

  # We add this rule because allowing traffic to the AKS FQDN tag actually includes `storage.googleapis.com` as well (at least at time of writing).
  # In order to simulate environment where this is not allowed by default (as CAST AI depends on it), we explicitly forbid it.
  # The list of URLs behind the tag is not public and dynamic; this was found via trial and error.
  rule {
    name             = "forbid google storage for testing fqdn for AKS"
    source_addresses = ["*"]
    target_fqdns     = ["storage.googleapis.com"]

    protocol {
      port = "443"
      type = "Https"
    }

    protocol {
      port = "80"
      type = "Http"
    }
  }
}

resource "azurerm_firewall_application_rule_collection" "fw-aks" {
  name                = "application-rules"
  azure_firewall_name = azurerm_firewall.egress_firewall.name
  resource_group_name = azurerm_resource_group.rg.name
  priority            = 400
  action              = "Allow"

  rule {
    name             = "allow fqdn for AKS"
    source_addresses = ["10.42.1.0/24"]
    fqdn_tags        = ["AzureKubernetesService"]
  }
}
