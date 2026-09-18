# Example: Azure external connections.
#
# This configuration is for reference only — do not apply without reviewing the
# placeholder values below. It demonstrates the main interaction patterns of the
# castai_external_connection / castai_external_connection_principals resources:
#
#   1. Multiple subscriptions connected with for_each (one principals resource +
#      one connection per subscription — one scope_key each).
#   2. Per-subscription feature sets and per-feature Entra app registrations.
#   3. Sub-feature selection and an optional pinned registry_version.
#
# Flow for each connection:
#   1. castai_external_connection_principals provisions CAST-side IAM principals
#      (service principals, roles) and emits a resource_suffix output.
#   2. castai_external_connection creates/updates the connection, passing that
#      resource_suffix back to CAST AI along with the customer-side Azure Entra
#      app registrations (one app per feature).

terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 7.0.0"
    }
  }
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

variable "castai_api_url" {
  description = "CAST AI API URL."
  type        = string
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  description = "CAST AI API token. Replace with your actual token."
  type        = string
  default     = "replace-with-your-api-token"
}

variable "azure_tenant_id" {
  description = "Azure Entra tenant ID for the app registrations."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "azure_subscriptions" {
  description = <<-EOT
    Map of Azure subscriptions to connect. Key is an arbitrary label; each entry
    contains the subscription ID (scope_key), the features to enable, and the
    per-feature Entra app registration client IDs created in the customer tenant.
    Feature keys must match the feature enum values.
  EOT
  type = map(object({
    subscription_id = string
    features        = list(string)
    apps            = map(string)
  }))
  default = {
    prod = {
      subscription_id = "00000000-0000-0000-0000-000000000100"
      features = [
        "NODE_AUTOSCALING",
        "WORKLOAD_AUTOSCALING",
        "COST_MONITORING",
      ]
      apps = {
        NODE_AUTOSCALING     = "00000000-0000-0000-0000-000000000001"
        WORKLOAD_AUTOSCALING = "00000000-0000-0000-0000-000000000002"
        COST_MONITORING      = "00000000-0000-0000-0000-000000000003"
      }
    }
    nonprod = {
      subscription_id = "00000000-0000-0000-0000-000000000200"
      features = [
        "NODE_AUTOSCALING",
        "KARPENTER_ENTERPRISE",
      ]
      apps = {
        NODE_AUTOSCALING    = "00000000-0000-0000-0000-000000000004"
        KARPENTER_ENTERPRISE = "00000000-0000-0000-0000-000000000005"
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Per-subscription principals + connections (one scope_key per subscription).
# -----------------------------------------------------------------------------

# CAST-side IAM principals for each subscription. The resource_suffix output is
# passed to the corresponding castai_external_connection resource.
resource "castai_external_connection_principals" "this" {
  for_each = var.azure_subscriptions

  cloud_provider   = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = each.value.subscription_id

  # One features block per enabled feature. Sub-features and registry_version
  # are optional; see the "cost_detailed" subscription below for an example of
  # explicit blocks instead of a dynamic loop.
  dynamic "features" {
    for_each = each.value.features
    content {
      feature = features.value
    }
  }
}

# Connection linking each Azure subscription to CAST AI. The resource_suffix
# from the principals resource establishes the trust relationship. Each enabled
# feature requires its own Entra app registration (client_id + tenant_id).
resource "castai_external_connection" "this" {
  for_each = var.azure_subscriptions

  cloud            = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = each.value.subscription_id
  resource_suffix  = castai_external_connection_principals.this[each.key].resource_suffix

  dynamic "enabled_features" {
    for_each = each.value.features
    content {
      feature = enabled_features.value
    }
  }

  metadata {
    azure {
      subscription_id = each.value.subscription_id

      dynamic "apps" {
        for_each = each.value.apps
        content {
          feature   = apps.key
          client_id = apps.value
          tenant_id = var.azure_tenant_id
        }
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Single explicit connection with sub-features and a pinned registry_version.
# This shows the fully expanded block form — useful for reviewing the raw
# interaction without dynamic blocks.
# -----------------------------------------------------------------------------

variable "azure_detailed_subscription_id" {
  description = "Azure subscription for the explicit sub-feature demo connection."
  type        = string
  default     = "00000000-0000-0000-0000-000000000300"
}

variable "azure_detailed_apps" {
  description = "Entra app client IDs for the detailed demo subscription."
  type        = map(string)
  default = {
    COST_MONITORING = "00000000-0000-0000-0000-000000000006"
  }
}

resource "castai_external_connection_principals" "detailed" {
  cloud_provider   = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_detailed_subscription_id

  features {
    feature = "COST_MONITORING"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
    sub_features = [
      "COST_MONITORING_STORAGE_METRICS",
      "COST_MONITORING_NETWORK_MONITORING",
      "COST_MONITORING_NETWORK_CONTEXT",
    ]
  }
}

resource "castai_external_connection" "detailed" {
  cloud            = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_detailed_subscription_id
  resource_suffix  = castai_external_connection_principals.detailed.resource_suffix

  enabled_features {
    feature = "COST_MONITORING"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
    sub_features = [
      "COST_MONITORING_STORAGE_METRICS",
      "COST_MONITORING_NETWORK_MONITORING",
      "COST_MONITORING_NETWORK_CONTEXT",
    ]
  }

  metadata {
    azure {
      subscription_id = var.azure_detailed_subscription_id

      apps {
        feature   = "COST_MONITORING"
        client_id = var.azure_detailed_apps["COST_MONITORING"]
        tenant_id = var.azure_tenant_id
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Outputs for inspection after apply.
# -----------------------------------------------------------------------------

output "connection_ids" {
  description = "IDs of the Azure external connections, keyed by subscription label."
  value = {
    for k, conn in castai_external_connection.this : k => conn.id
  }
}

output "detailed_connection_id" {
  description = "ID of the explicit sub-feature demo connection."
  value        = castai_external_connection.detailed.id
}

output "principals_resource_suffixes" {
  description = "Resource suffixes generated per subscription, keyed by label."
  value = {
    for k, p in castai_external_connection_principals.this : k => p.resource_suffix
  }
}
