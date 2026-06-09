# 1. Create virtual network and resource group for the cluster.

resource "azurerm_resource_group" "this" {
  name     = var.cluster_name
  location = var.cluster_region
}

resource "azurerm_virtual_network" "this" {
  name                = "${var.cluster_name}-network"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  address_space       = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "internal" {
  name                 = "internal"
  virtual_network_name = azurerm_virtual_network.this.name
  resource_group_name  = azurerm_resource_group.this.name
  address_prefixes     = ["10.1.0.0/22"]
}

# 2. Create AKS cluster.

resource "azurerm_kubernetes_cluster" "this" {
  name                = var.cluster_name
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  dns_prefix          = var.cluster_name
  node_resource_group = "${var.cluster_name}-ng"
  kubernetes_version  = var.cluster_version

  default_node_pool {
    name = "default"
    # Node count has to be > 2 to successfully deploy CAST AI controller.
    node_count     = 2
    vm_size        = "Standard_D2_v2"
    vnet_subnet_id = azurerm_subnet.internal.id
  }

  identity {
    type = "SystemAssigned"
  }

  tags = {
    Environment = "Test"
  }
}

# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes = {
    host                   = azurerm_kubernetes_cluster.this.kube_config.0.host
    client_certificate     = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_key)
    cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.cluster_ca_certificate)
  }
}

# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  subscription_id = var.subscription_id
  features {}
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

# 3. Connect AKS cluster to CAST AI and configure live migration.

module "cluster" {
  source = "./module/castai"
  count  = var.enable_castai ? 1 : 0

  cluster_name     = var.cluster_name
  cluster_region   = var.cluster_region
  castai_api_token = var.castai_api_token
  castai_api_url   = var.castai_api_url
  castai_grpc_url  = var.castai_grpc_url

  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name
  subscription_id     = data.azurerm_subscription.current.subscription_id
  tenant_id           = data.azurerm_subscription.current.tenant_id
  subnet_id           = azurerm_subnet.internal.id

  tags                       = var.tags
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  install_helm_live          = var.install_helm_live
  live_helm_version          = var.live_helm_version

  depends_on = [azurerm_kubernetes_cluster.this]
}
