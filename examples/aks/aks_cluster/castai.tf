# 3. Connect AKS cluster to CAST AI in READ-ONLY mode.

# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.this.kube_config.0.host
    client_certificate      = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_key)
    cluster_ca_certificate  = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.cluster_ca_certificate)
  }
}

# Configure AKS cluster connection to CAST AI using CAST AI aks-cluster module.
module "castai-aks-cluster" {
  source = "castai/aks/castai"

  api_url             = var.castai_api_url

  aks_cluster_name    = var.cluster_name
  aks_cluster_region  = var.cluster_region
  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [azurerm_subnet.internal.id]
      tags           = var.tags
    }
  }
}
