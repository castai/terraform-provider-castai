locals {
  http_proxy_address  = "${azurerm_firewall.explicit_firewall.ip_configuration[0].private_ip_address}:${azurerm_firewall_policy.explicit_proxy_policy.explicit_proxy[0].http_port}"
  https_proxy_address = "${azurerm_firewall.explicit_firewall.ip_configuration[0].private_ip_address}:${azurerm_firewall_policy.explicit_proxy_policy.explicit_proxy[0].https_port}"
  no_proxy_default    = concat([azurerm_kubernetes_cluster.aks.fqdn], var.fqdn_without_proxy)
  no_proxy_agent      = concat(local.no_proxy_default, ["169.254.169.254"]) # Agent requires access to local metadata endpoint as well
}

module "castai-aks-cluster" {
  source = "castai/aks/castai"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aks_cluster_name    = var.cluster_name
  aks_cluster_region  = azurerm_kubernetes_cluster.aks.location
  node_resource_group = azurerm_kubernetes_cluster.aks.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.aks.resource_group_name

  http_proxy  = local.http_proxy_address
  https_proxy = local.https_proxy_address
  no_proxy    = var.fqdn_without_proxy

  delete_nodes_on_disconnect = true

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [azurerm_subnet.aks_subnet.id]
    }
  }

  # Configure proxy and disable VPA. The VPA image source is dynamic and hard to make a static "firewall" rule for - this should make running the example easier.
  agent_values = [
    <<-EOF
    podAnnotations:
      kubernetes.azure.com/set-kube-service-host-fqdn: "true"
    additionalEnv:
      HTTP_PROXY: "${local.http_proxy_address}"
      HTTPS_PROXY: "${local.https_proxy_address}"
      NO_PROXY: "${join(",", local.no_proxy_agent)}"
    clusterVPA:
      enabled: false
    EOF
  ]

  cluster_controller_values = [
    <<-EOF
    podAnnotations:
      kubernetes.azure.com/set-kube-service-host-fqdn: "true"
    additionalEnv:
      HTTP_PROXY: "${local.http_proxy_address}"
      HTTPS_PROXY: "${local.https_proxy_address}"
      NO_PROXY: "${join(",", local.no_proxy_default)}"
    EOF
  ]

  # Networking setup should be completed before trying to onboard cluster; otherwise components get stuck with no connectivity.
  depends_on = [
    azurerm_firewall.egress_firewall,
    azurerm_firewall.explicit_firewall,
    azurerm_firewall_network_rule_collection.egress_rule
  ]
}