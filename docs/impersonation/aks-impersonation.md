# Impersonation for AKS clusters

Cast AI supports AKS Workload Identity Impersonation as an alternative to using App Registration passwords. This method enhances security by eliminating the need for password management while still allowing Cast AI to access your Azure resources.

For general information about this method, see https://docs.cast.ai/docs/sa-impersonation-aks.

This document describes how to configure AKS Impersonation using Terraform for new clusters and how to migrate existing clusters to Impersonation.

## What am I using now?

If your `castai_aks_cluster` resource has `client_secret` set, you're using App Registration password (static credentials). You should see an `azuread_application_password` resource in your Terraform state.

If your `castai_aks_cluster` resource has `federation_id` set, you're using Workload Identity Impersonation. You should see an `azurerm_federated_identity_credential` resource in your Terraform state.

## Connecting with Impersonation

If you want to connect a new cluster to Cast AI, set `authentication_method = "workload_identity"` on the `castai/aks` module. Module versions `13.0+` default to this `authentication_method`.

## Migrating from App Registration password to Workload Identity Impersonation

If your cluster is connected using App Registration password (static credentials), you can migrate it to Impersonation. This involves the following three steps:
1. Set up Azure resources required for Impersonation to establish trust between your Azure Managed Identity and Cast AI.
2. Switch your cluster to use the new authentication method.
3. Clean up Azure resources that are no longer required. Cast AI will no longer use your App Registration, so it can be removed.

The following section describes some prerequisites, and the ones after that describe these steps in detail.

### Prerequisites

- Cast AI provider **>= 9.14.0**:
  ```hcl
  terraform {
    required_providers {
      castai = {
        source  = "castai/castai"
        version = ">= 9.14.0"
      }
    }
  }
  ```

### 1. Setting up Impersonation Azure resources

If you **use** the `castai/aks` module, update it to version `~> 12.1` (do _not_ update to `13.x` yet) and set the following arguments:

```terraform
module "castai_aks_cluster" {
  source  = "castai/aks/castai"
  # Change version to `~> 12.1`
  version = "~> 12.1"

  # Set these two arguments. Note that these are the defaults in ~> 12.1. If you've
  # already updated to that version and haven't changed them, you're good.
  authentication_method           = "client_secret"
  authentication_method_migration = "prepare"
}
```

You should see the following changes:
- Adding `module.castai_aks_cluster.azurerm_user_assigned_identity.this`. This is the Managed Identity that Cast AI will impersonate.
- Adding `module.castai_aks_cluster.azurerm_federated_identity_credential.this`. This configures the federation that allows Cast AI to impersonate the Managed Identity.

If you **don't use** the `castai/aks` module, you need to create an `azurerm_user_assigned_identity` and `azurerm_federated_identity_credential` to define the Managed Identity and allow Cast AI to impersonate it:

```terraform
# Define castai_impersonation_service_account data source
data "castai_impersonation_service_account" "this" {
  provider_type = "aks"
}

# Create azurerm_user_assigned_identity
resource "azurerm_user_assigned_identity" "this" {
  name                = "${var.aks_cluster_name}-castai-identity" # Suggested name
  resource_group_name = var.resource_group
  location            = data.azurerm_kubernetes_cluster.castai.location
}

# Create azurerm_federated_identity_credential
resource "azurerm_federated_identity_credential" "this" {
  name                      = substr("castai-${var.aks_cluster_name}-${var.resource_group}", 0, 64) # Suggested name
  audience                  = ["api://AzureADTokenExchange"]
  issuer                    = "https://accounts.google.com"
  user_assigned_identity_id = azurerm_user_assigned_identity.this.id
  subject                   = data.castai_impersonation_service_account.this.id
}
```

### 2. Switch the authentication method

After verifying that the previous step was successfully applied and the `azurerm_user_assigned_identity` and `azurerm_federated_identity_credential` resources were created, you can switch the authentication method from using the old App Registration password to Impersonation.

While applying this change, Cast AI will briefly be unable to access your cloud resources, and thus **there can be a gap in all Cast AI services during this period**. Schedule this operation in a maintenance window to minimize the impact.

If you **use** the `castai/aks` module, update to version `~> 13.0` and change the following arguments:

```terraform
module "castai_aks_cluster" {
  source  = "castai/aks/castai"
  # Change version to `~> 13.0`
  version = "~> 13.0"

  # Change `authentication_method` to `workload_identity`
  authentication_method = "workload_identity"
  # Change `authentication_method_migration` to `keep-secret-auth` so App Registration is not cleaned up yet.
  authentication_method_migration = "keep-secret-auth"
}
```

You should see the following changes:
- Replacing `module.castai_aks_cluster.azurerm_role_assignment.castai_node_resource_group` to assign this role to the Managed Identity instead of the App Registration.
- Replacing `module.castai_aks_cluster.azurerm_role_assignment.castai_resource_group` to assign this role to the Managed Identity instead of the App Registration.
- Updating `module.castai_aks_cluster.castai_aks_cluster.castai_cluster` to remove `client_secret`, add `federation_id` and change `client_id`.

If you **don't use** the `castai/aks` module, you need to change `azurerm_role_assignment`s to point to the Managed Identity and update the inputs of `castai_aks_cluster`:

```terraform
resource "azurerm_role_assignment" "castai_resource_group" {
  # `principal_id` previously pointed to an `azuread_service_principal` like this:
  # principal_id  = azuread_service_principal.castai.object_id
  # That needs to be updated to:
  principal_id = azurerm_user_assigned_identity.this.principal_id
  
  # The following arguments can remain the same as before. These are examples of
  # what they probably looked like:
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id
  description        = "castai role assignment for resource group ${var.resource_group}"
  scope              = "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group}"
}

resource "azurerm_role_assignment" "castai_node_resource_group" {
  # `principal_id` previously pointed to an `azuread_service_principal` like this:
  # principal_id = azuread_service_principal.castai.object_id
  # That needs to be updated to:
  principal_id       = azurerm_user_assigned_identity.this.principal_id

  # The following arguments can remain the same as before. These are examples on
  # what they probably looked like:
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id
  description        = "castai role assignment for resource group ${var.aks_cluster_name}"
  scope              = "/subscriptions/${var.subscription_id}/resourceGroups/${var.node_resource_group}"
}

# Note: If you had any other `azurerm_role_assignment`s defined for `azuread_service_principal.castai`, you need to update them the same way, changing the `principal_id` to `azurerm_user_assigned_identity.this.principal_id`!

# Change the following arguments of `castai_aks_cluster`:
resource "castai_aks_cluster" "castai_cluster" {
  # Update `client_id` to point to the new `azurerm_user_assigned_identity`:
  client_id = azurerm_user_assigned_identity.this.client_id
  
  # Remove `client_secret`:
  # client_secret = azuread_application_password.castai.value
  
  # Set `federation_id` to point to the new `azurerm_user_assigned_identity`:
  federation_id = azurerm_user_assigned_identity.this.principal_id
  
  # ...
}
```

### 3. Clean up App Registration

After applying the previous changes, your App Registration is no longer in use by Cast AI, so you can remove it.

If you **use** the `castai/aks` module, change the following arguments:

```terraform
module "castai_aks_cluster" {
  source  = "castai/aks/castai"
  version = "~> 13.0"
  
  # Remove `authentication_method_migration`:
  # authentication_method_migration = "keep-secret-auth"
}
```

You should see the following changes:
- Destroying `module.castai_aks_cluster.azuread_application.castai`.
- Destroying `module.castai_aks_cluster.azuread_application_password.castai`.
- Destroying `module.castai_aks_cluster.azuread_service_principal.castai`.

If you **don't use** the `castai/aks` module, remove the following resources:

```terraform
# Remove `azuread_application`:
# resource "azuread_application" "castai" {
#   # ...
# }

# Remove `azuread_application_password`:
# resource "azuread_application_password" "castai" {
#   application_id = azuread_application.castai.id
# }

# Remove `azuread_service_principal`:
# resource "azuread_service_principal" "castai" {
#   client_id = azuread_application.castai.client_id
#   # ...
# }
```