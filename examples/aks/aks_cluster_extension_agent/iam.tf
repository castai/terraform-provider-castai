# 4. Cast AI needed cloud permissions.

locals {
  role_name = "CastAKSRole-${var.cluster_name}-tf"
  app_name  = substr("CAST AI ${var.cluster_name}-${var.resource_group}", 0, 64)
}

// Azure RM
resource "azurerm_role_definition" "castai" {
  name        = local.role_name
  description = "Role used by CAST AI"

  scope = "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group}"

  permissions {
    actions = [
      "Microsoft.Compute/*/read",
      "Microsoft.Compute/virtualMachines/*",
      "Microsoft.Compute/virtualMachineScaleSets/*",
      "Microsoft.Compute/disks/write",
      "Microsoft.Compute/disks/delete",
      "Microsoft.Compute/disks/beginGetAccess/action",
      "Microsoft.Compute/galleries/write",
      "Microsoft.Compute/galleries/delete",
      "Microsoft.Compute/galleries/images/write",
      "Microsoft.Compute/galleries/images/delete",
      "Microsoft.Compute/galleries/images/versions/write",
      "Microsoft.Compute/galleries/images/versions/delete",
      "Microsoft.Compute/snapshots/write",
      "Microsoft.Compute/snapshots/delete",
      "Microsoft.Network/*/read",
      "Microsoft.Network/networkInterfaces/write",
      "Microsoft.Network/networkInterfaces/delete",
      "Microsoft.Network/networkInterfaces/join/action",
      "Microsoft.Network/networkSecurityGroups/join/action",
      "Microsoft.Network/virtualNetworks/subnets/join/action",
      "Microsoft.Network/applicationGateways/backendhealth/action",
      "Microsoft.Network/applicationGateways/backendAddressPools/join/action",
      "Microsoft.Network/applicationSecurityGroups/joinIpConfiguration/action",
      "Microsoft.Network/loadBalancers/backendAddressPools/write",
      "Microsoft.Network/loadBalancers/backendAddressPools/join/action",
      "Microsoft.ContainerService/*/read",
      "Microsoft.ContainerService/managedClusters/start/action",
      "Microsoft.ContainerService/managedClusters/stop/action",
      "Microsoft.ContainerService/managedClusters/runCommand/action",
      "Microsoft.ContainerService/managedClusters/agentPools/*",
      "Microsoft.Resources/*/read",
      "Microsoft.Resources/tags/write",
      "Microsoft.Authorization/locks/read",
      "Microsoft.Authorization/roleAssignments/read",
      "Microsoft.Authorization/roleDefinitions/read",
      "Microsoft.ManagedIdentity/userAssignedIdentities/assign/action",
    ]
    not_actions = []
  }

  assignable_scopes = distinct(compact(flatten([
    "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group}",
    "/subscriptions/${var.subscription_id}/resourceGroups/${azurerm_kubernetes_cluster.this.node_resource_group}",
    var.additional_resource_groups
  ])))

  depends_on = [azurerm_kubernetes_cluster.this]
}

resource "azurerm_role_assignment" "castai_resource_group" {
  principal_id       = azuread_service_principal.castai.object_id
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id
  description        = "castai role assignment for resource group ${var.resource_group}"
  scope              = "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group}"
}

resource "azurerm_role_assignment" "castai_node_resource_group" {
  principal_id       = azuread_service_principal.castai.object_id
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id
  description        = "castai role assignment for resource group ${var.cluster_name}"
  scope              = "/subscriptions/${var.subscription_id}/resourceGroups/${azurerm_kubernetes_cluster.this.node_resource_group}"
}

resource "azurerm_role_assignment" "castai_additional_resource_groups" {
  for_each           = toset(var.additional_resource_groups)
  principal_id       = azuread_service_principal.castai.object_id
  description        = "castai role assignment for resource group ${each.key}"
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id
  scope              = each.key
}

// Azure AD

data "azuread_client_config" "current" {}

resource "azuread_application" "castai" {
  display_name = local.app_name
  owners       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application_password" "castai" {
  application_id = azuread_application.castai.id
}

resource "azuread_service_principal" "castai" {
  client_id                    = azuread_application.castai.client_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}
