# Example: AWS external connections.
#
# This configuration is for reference only — do not apply without reviewing the
# placeholder values below. It demonstrates the main interaction patterns of the
# castai_external_connection / castai_external_connection_principals resources:
#
#   1. A "full" connection on the primary AWS account: NODE_AUTOSCALING with
#      sub-features, WORKLOAD_AUTOSCALING, and COST_MONITORING.
#   2. A second connection on a *different* AWS account (new scope_key) with a
#      smaller feature set and a pinned registry_version.
#   3. (Commented) a second connection on the *same* AWS account — the
#      multi-connection-per-account scenario enabled by the optional
#      scope_key argument on castai_external_connection_principals.
#
# Flow for each connection:
#   1. castai_external_connection_principals provisions CAST-side IAM principals
#      and emits a resource_suffix output.
#   2. castai_external_connection creates/updates the connection, passing that
#      resource_suffix back to CAST AI along with the customer-side IAM role ARN.

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

variable "aws_account_id" {
  description = "Primary AWS account ID to connect to CAST AI."
  type        = string
  default     = "123456789012"
}

variable "aws_role_arn" {
  description = "IAM role ARN in the primary AWS account that CAST AI assumes."
  type        = string
  default     = "arn:aws:iam::123456789012:role/castai-external-connection"
}

variable "aws_secondary_account_id" {
  description = "Second AWS account connected to CAST AI (new scope_key demo)."
  type        = string
  default     = "210987654321"
}

variable "aws_secondary_role_arn" {
  description = "IAM role ARN in the second AWS account that CAST AI assumes."
  type        = string
  default     = "arn:aws:iam::210987654321:role/castai-external-connection"
}

# -----------------------------------------------------------------------------
# Connection 1: primary AWS account, full feature set.
# -----------------------------------------------------------------------------

# CAST-side IAM principals for the primary account. Sub-features are selected per
# feature; empty sub_features means base permissions only.
resource "castai_external_connection_principals" "primary" {
  cloud_provider   = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_account_id

  features {
    feature = "NODE_AUTOSCALING"
    sub_features = [
      "NODE_AUTOSCALING_SPOT_INTERRUPTION_HANDLING",
    ]
  }

  features {
    feature = "WORKLOAD_AUTOSCALING"
    sub_features = [
      "WORKLOAD_AUTOSCALING_GPU_AWARE_SCALING",
    ]
  }

  features {
    feature = "COST_MONITORING"
    sub_features = [
      "COST_MONITORING_STORAGE_METRICS",
      "COST_MONITORING_NETWORK_MONITORING",
    ]
  }
}

# Connection linking the primary AWS account to CAST AI. The resource_suffix from
# the principals resource establishes the trust relationship.
resource "castai_external_connection" "primary" {
  cloud            = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_account_id
  resource_suffix  = castai_external_connection_principals.primary.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
    sub_features = [
      "NODE_AUTOSCALING_SPOT_INTERRUPTION_HANDLING",
    ]
  }

  enabled_features {
    feature = "WORKLOAD_AUTOSCALING"
    sub_features = [
      "WORKLOAD_AUTOSCALING_GPU_AWARE_SCALING",
    ]
  }

  enabled_features {
    feature = "COST_MONITORING"
    sub_features = [
      "COST_MONITORING_STORAGE_METRICS",
      "COST_MONITORING_NETWORK_MONITORING",
    ]
  }

  metadata {
    aws {
      role_arn = var.aws_role_arn
    }
  }
}

# -----------------------------------------------------------------------------
# Connection 2: secondary AWS account (new scope_key), smaller feature set and a
# pinned registry_version (permissions snapshot instead of latest).
# -----------------------------------------------------------------------------

resource "castai_external_connection_principals" "secondary_account" {
  cloud_provider   = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_secondary_account_id

  features {
    feature = "NODE_AUTOSCALING"
  }

  features {
    feature = "KARPENTER_ENTERPRISE"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
  }
}

resource "castai_external_connection" "secondary_account" {
  cloud            = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_secondary_account_id
  resource_suffix  = castai_external_connection_principals.secondary_account.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
  }

  enabled_features {
    feature = "KARPENTER_ENTERPRISE"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
  }

  metadata {
    aws {
      role_arn = var.aws_secondary_role_arn
    }
  }
}

# -----------------------------------------------------------------------------
# (Optional) Connection 3: second connection on the SAME AWS account.
#
# The optional scope_key on castai_external_connection_principals exists for
# multi-connection-per-account scenarios: uniqueness becomes
# (organization_id, cloud_provider, scope_key) instead of
# (organization_id, cloud_provider), so each scoped principals resource gets its
# own resource_suffix. Uncomment to try this interaction.
# -----------------------------------------------------------------------------

# resource "castai_external_connection_principals" "primary_secondary_connection" {
#   cloud_provider   = "AWS"
#   connection_scope = "AWS_ACCOUNT"
#   scope_key        = "${var.aws_account_id}/cost-only"
#
#   features {
#     feature = "COST_MONITORING"
#   }
# }

# resource "castai_external_connection" "primary_secondary_connection" {
#   cloud            = "AWS"
#   connection_scope = "AWS_ACCOUNT"
#   scope_key        = var.aws_account_id
#   resource_suffix  = castai_external_connection_principals.primary_secondary_connection.resource_suffix
#
#   enabled_features {
#     feature = "COST_MONITORING"
#   }
#
#   metadata {
#     aws {
#       role_arn = var.aws_role_arn
#     }
#   }
# }

# -----------------------------------------------------------------------------
# Outputs for inspection after apply.
# -----------------------------------------------------------------------------

output "primary_connection_id" {
  description = "ID of the primary AWS external connection."
  value        = castai_external_connection.primary.id
}

output "secondary_account_connection_id" {
  description = "ID of the secondary AWS account external connection."
  value        = castai_external_connection.secondary_account.id
}

output "connections" {
  description = "All AWS external connections created by this configuration."
  value = {
    primary           = castai_external_connection.primary
    secondary_account = castai_external_connection.secondary_account
  }
}
