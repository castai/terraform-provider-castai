# Example: Azure external connection enabling node autoscaling, workload autoscaling (woop),
# and cost monitoring. This configuration is for reference only — do not apply without
# reviewing the placeholder values below.
#
# Flow:
#   1. castai_external_connection_principals provisions CAST-side IAM principals and
#      emits a resource_suffix output.
#   2. castai_external_connection creates the connection, passing that resource_suffix
#      back to CAST AI along with the customer-side Azure Entra app registrations.

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

variable "azure_subscription_id" {
  description = "Azure subscription ID to connect to CAST AI."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "azure_tenant_id" {
  description = "Azure Entra tenant ID for the app registrations."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "azure_node_autoscaling_client_id" {
  description = "Entra app (application) client ID for the NODE_AUTOSCALING feature."
  type        = string
  default     = "00000000-0000-0000-0000-000000000001"
}

variable "azure_workload_autoscaling_client_id" {
  description = "Entra app (application) client ID for the WORKLOAD_AUTOSCALING feature."
  type        = string
  default     = "00000000-0000-0000-0000-000000000002"
}

variable "azure_cost_monitoring_client_id" {
  description = "Entra app (application) client ID for the COST_MONITORING feature."
  type        = string
  default     = "00000000-0000-0000-0000-000000000003"
}

# Provision CAST-side IAM principals (service principals, roles) for the requested features.
# The resource_suffix output is passed to the castai_external_connection resource.
resource "castai_external_connection_principals" "this" {
  cloud_provider   = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_subscription_id

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

# Create the external connection linking the Azure subscription to CAST AI.
# The resource_suffix from the principals resource establishes the trust relationship.
# Each feature requires its own Entra app registration (client_id + tenant_id).
resource "castai_external_connection" "this" {
  cloud            = "AZURE"
  connection_scope = "AZURE_SUBSCRIPTION"
  scope_key        = var.azure_subscription_id
  resource_suffix  = castai_external_connection_principals.this.resource_suffix

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
      subscription_id = var.azure_subscription_id

      apps {
        feature   = "NODE_AUTOSCALING"
        client_id = var.azure_node_autoscaling_client_id
        tenant_id = var.azure_tenant_id
      }

      apps {
        feature   = "WORKLOAD_AUTOSCALING"
        client_id = var.azure_workload_autoscaling_client_id
        tenant_id = var.azure_tenant_id
      }

      apps {
        feature   = "COST_MONITORING"
        client_id = var.azure_cost_monitoring_client_id
        tenant_id = var.azure_tenant_id
      }
    }
  }
}
