# 3. Accept free plan terms and create azure extension that triggers read only agent deployment.

# NOTE: When plan is specified, legal terms must be accepted for this item on this subscription before creating the Kubernetes Cluster Extension.
# The azurerm_marketplace_agreement resource or AZ CLI tool can be used to do this.
resource "azurerm_marketplace_agreement" "accept_terms" {
  publisher = "castaigroupinc1683643265413"
  offer     = "castai-agent"
  plan      = "free_plan"
}

resource "azurerm_kubernetes_cluster_extension" "castai" {
  name              = "castai-agent"
  cluster_id        = azurerm_kubernetes_cluster.this.id
  extension_type    = "CASTAI.agent"
  release_namespace = "castai-agent"
  configuration_settings = {
    provider = "aks"
    apiKey   = var.castai_api_token
  }

  plan {
    publisher = "castaigroupinc1683643265413"
    product   = "castai-agent"
    name      = "free_plan"
  }
  depends_on = [azurerm_marketplace_agreement.accept_terms]
}
