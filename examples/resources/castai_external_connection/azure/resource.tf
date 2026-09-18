# Example: Azure external connections.
#
# This configuration is for reference only — do not apply without reviewing the
# placeholder values below. It demonstrates the main interaction patterns of the
# castai_external_connection / castai_external_connection_principals resources:
#
#   1. A "full" connection on the primary subscription: NODE_AUTOSCALING,
#      WORKLOAD_AUTOSCALING, and COST_MONITORING, each with its own Entra app
#      registration.
#   2. A second connection on a *different* subscription (new scope_key) with a
#      smaller feature set.
#   3. A third connection with sub-feature selection and an optional pinned
#      registry_version.
#
# Everything is written out explicitly on purpose — duplication is fine here,
# the goal is to show exactly what the Terraform interaction looks like.
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

# -----------------------------------------------------------------------------
# Connection 1: primary subscription, full feature set.
# -----------------------------------------------------------------------------

variable "azure_primary_subscription_id" {
  description = "Primary Azure subscription ID to connect to CAST AI."
  type        = string
  default     = "00000000-0000-0000-0000-000000000100"
}

# CAST-side IAM principals for the primary subscription. The resource_suffix
# output is passed to the castai_external_connection resource.
resource "castai_external_connection_principals" "primary" {
  cloud_provider   = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_primary_subscription_id

  features {
    feature = "NODE_AUTOSCALING"
  }

  features {
    feature = "WORKLOAD_AUTOSCALING"
  }

  features {
    feature = "COST_MONITORING"
  }
}

# Connection linking the primary Azure subscription to CAST AI. The
# resource_suffix from the principals resource establishes the trust
# relationship. Each enabled feature requires its own Entra app registration
# (client_id + tenant_id).
resource "castai_external_connection" "primary" {
  cloud            = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_primary_subscription_id
  resource_suffix  = castai_external_connection_principals.primary.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
  }

  enabled_features {
    feature = "WORKLOAD_AUTOSCALING"
  }

  enabled_features {
    feature = "COST_MONITORING"
  }

  metadata {
    azure {
      subscription_id = var.azure_primary_subscription_id

      apps {
        feature   = "NODE_AUTOSCALING"
        client_id = "00000000-0000-0000-0000-000000000001"
        tenant_id = var.azure_tenant_id
      }

      apps {
        feature   = "WORKLOAD_AUTOSCALING"
        client_id = "00000000-0000-0000-0000-000000000002"
        tenant_id = var.azure_tenant_id
      }

      apps {
        feature   = "COST_MONITORING"
        client_id = "00000000-0000-0000-0000-000000000003"
        tenant_id = var.azure_tenant_id
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Connection 2: secondary subscription (new scope_key), smaller feature set.
# -----------------------------------------------------------------------------

variable "azure_secondary_subscription_id" {
  description = "Second Azure subscription connected to CAST AI (new scope_key demo)."
  type        = string
  default     = "00000000-0000-0000-0000-000000000200"
}

resource "castai_external_connection_principals" "secondary_subscription" {
  cloud_provider   = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_secondary_subscription_id

  features {
    feature = "NODE_AUTOSCALING"
  }

  features {
    feature = "KARPENTER_ENTERPRISE"
  }
}

resource "castai_external_connection" "secondary_subscription" {
  cloud            = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_secondary_subscription_id
  resource_suffix  = castai_external_connection_principals.secondary_subscription.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
  }

  enabled_features {
    feature = "KARPENTER_ENTERPRISE"
  }

  metadata {
    azure {
      subscription_id = var.azure_secondary_subscription_id

      apps {
        feature   = "NODE_AUTOSCALING"
        client_id = "00000000-0000-0000-0000-000000000004"
        tenant_id = var.azure_tenant_id
      }

      apps {
        feature   = "KARPENTER_ENTERPRISE"
        client_id = "00000000-0000-0000-0000-000000000005"
        tenant_id = var.azure_tenant_id
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Connection 3: sub-feature selection and an optional pinned registry_version.
# -----------------------------------------------------------------------------

variable "azure_detailed_subscription_id" {
  description = "Azure subscription for the sub-feature demo connection."
  type        = string
  default     = "00000000-0000-0000-0000-000000000300"
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
        client_id = "00000000-0000-0000-0000-000000000006"
        tenant_id = var.azure_tenant_id
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Outputs for inspection after apply.
# -----------------------------------------------------------------------------

output "primary_connection_id" {
  description = "ID of the primary Azure external connection."
  value        = castai_external_connection.primary.id
}

output "secondary_subscription_connection_id" {
  description = "ID of the secondary subscription external connection."
  value        = castai_external_connection.secondary_subscription.id
}

output "detailed_connection_id" {
  description = "ID of the sub-feature demo connection."
  value        = castai_external_connection.detailed.id
}
