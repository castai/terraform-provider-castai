locals {
  # Regexes from https://developer.hashicorp.com/terraform/language/functions/regex#examples
  castai_api_url_parts = regex("(?:(?P<scheme>[^:/?#]+):)?(?://(?P<authority>[^/?#]*))?(?P<path>[^?#]*)(?:\\?(?P<query>[^#]*))?(?:#(?P<fragment>.*))?", var.castai_api_url)
  castai_api_fqdn      = local.castai_api_url_parts["authority"]
}

resource "azurerm_public_ip" "explicit_firewall_ip" {
  name                = "explicit-firewall-public-ip"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  allocation_method   = "Static"
  sku                 = "Standard"
}


resource "azurerm_firewall" "explicit_firewall" {
  name                = "explicit-proxy-firewall"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  sku_name            = "AZFW_VNet"
  sku_tier            = "Standard"

  ip_configuration {
    name                 = "configuration"
    subnet_id            = azurerm_subnet.explicit_firewall_subnet.id
    public_ip_address_id = azurerm_public_ip.explicit_firewall_ip.id
  }

  firewall_policy_id = azurerm_firewall_policy.explicit_proxy_policy.id

  lifecycle {
    ignore_changes = [
      tags["CreatedAt"]
    ]
  }
}


resource "azurerm_firewall_policy" "explicit_proxy_policy" {
  name                = "explicit-proxy-policy"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location

  explicit_proxy {
    enabled = true

    http_port  = 3128
    https_port = 3129
  }

  lifecycle {
    ignore_changes = [tags]
  }
}


resource "azurerm_firewall_policy_rule_collection_group" "explicit_proxy_rules" {
  name               = "allow-any-traffic"
  firewall_policy_id = azurerm_firewall_policy.explicit_proxy_policy.id
  priority           = 100

  network_rule_collection {
    name     = "allow-all"
    action   = "Allow"
    priority = 100

    rule {
      name                  = "allow-all"
      protocols             = ["Any"]
      source_addresses      = ["*"]
      destination_addresses = ["*"]
      destination_ports     = ["80", "443"]
    }
  }

  application_rule_collection {
    name     = "allow-traffic"
    action   = "Allow"
    priority = 200

    rule {
      name = "allow-cast"
      protocols {
        port = "80"
        type = "Http"
      }
      protocols {
        port = "443"
        type = "Https"
      }
      source_addresses = ["*"]
      destination_fqdns = [
        "storage.googleapis.com", # Storage required to pull custom CAST AI binaries from nodes.
        local.castai_api_fqdn
      ]
    }

    # Uncomment to allow all traffic
    # rule {
    #   name = "allow-all"
    #   protocols {
    #     port = "80"
    #     type = "Http"
    #   }
    #   protocols {
    #     port = "443"
    #     type = "Https"
    #   }
    #   source_addresses = ["*"]
    #   destination_fqdns = [ "*" ]
    # }
  }
}

