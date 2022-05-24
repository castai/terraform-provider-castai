provider "kubernetes" {
  host                   = azurerm_kubernetes_cluster.example.kube_config.0.host
  username               = azurerm_kubernetes_cluster.example.kube_config.0.username
  password               = azurerm_kubernetes_cluster.example.kube_config.0.password
  client_certificate     = base64decode(azurerm_kubernetes_cluster.example.kube_config.0.client_certificate)
  client_key             = base64decode(azurerm_kubernetes_cluster.example.kube_config.0.client_key)
  cluster_ca_certificate = azurerm_kubernetes_cluster.example.kube_config.0.client_certificate
}


resource "azurerm_resource_group" "example" {
  name     = var.aks_resource_group
  location = "West Europe"
}

resource "azurerm_kubernetes_cluster" "example" {
  name                = var.aks_cluster_name
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  dns_prefix          = var.aks_cluster_name

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_D2_v2"
  }

  identity {
    type = "SystemAssigned"
  }

  tags = {
    Environment = "Production"
  }
}