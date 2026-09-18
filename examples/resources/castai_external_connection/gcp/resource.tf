# Example: GCP external connections.
#
# This configuration is for reference only — do not apply without reviewing the
# placeholder values below. It demonstrates the main interaction patterns of the
# castai_external_connection / castai_external_connection_principals resources:
#
#   1. Multiple projects connected with for_each (one principals resource + one
#      connection per project — one scope_key each), with per-project feature
#      sets and per-feature service account emails.
#   2. An explicit connection showing sub-feature selection and an optional
#      pinned registry_version in the expanded block form.
#   3. (Commented) an org-level connection using the GCP_ORGANIZATION scope.
#
# Flow for each connection:
#   1. castai_external_connection_principals provisions CAST-side IAM principals
#      (service accounts, roles) and emits a resource_suffix output.
#   2. castai_external_connection creates/updates the connection, passing that
#      resource_suffix back to CAST AI along with the customer-side GCP service
#      account emails.

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

variable "gcp_projects" {
  description = <<-EOT
    Map of GCP projects to connect. Key is an arbitrary label; each entry
    contains the project ID (scope_key), the features to enable, and the
    per-feature customer-side service account emails created in the project.
  EOT
  type = map(object({
    project_id = string
    features   = list(string)
    service_account_emails = map(string)
  }))
  default = {
    prod = {
      project_id = "my-gcp-project-prod"
      features = [
        "NODE_AUTOSCALING",
        "WORKLOAD_AUTOSCALING",
        "COST_MONITORING",
      ]
      service_account_emails = {
        NODE_AUTOSCALING     = "node-autoscaling@my-gcp-project-prod.iam.gserviceaccount.com"
        WORKLOAD_AUTOSCALING = "workload-autoscaling@my-gcp-project-prod.iam.gserviceaccount.com"
        COST_MONITORING      = "cost-monitoring@my-gcp-project-prod.iam.gserviceaccount.com"
      }
    }
    nonprod = {
      project_id = "my-gcp-project-nonprod"
      features = [
        "NODE_AUTOSCALING",
        "KARPENTER_ENTERPRISE",
      ]
      service_account_emails = {
        NODE_AUTOSCALING    = "node-autoscaling@my-gcp-project-nonprod.iam.gserviceaccount.com"
        KARPENTER_ENTERPRISE = "karpenter@my-gcp-project-nonprod.iam.gserviceaccount.com"
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Per-project principals + connections (one scope_key per project).
# -----------------------------------------------------------------------------

# CAST-side IAM principals for each project. The resource_suffix output is
# passed to the corresponding castai_external_connection resource.
resource "castai_external_connection_principals" "this" {
  for_each = var.gcp_projects

  cloud_provider   = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = each.value.project_id

  dynamic "features" {
    for_each = each.value.features
    content {
      feature = features.value
    }
  }
}

# Connection linking each GCP project to CAST AI. The resource_suffix from the
# principals resource establishes the trust relationship.
resource "castai_external_connection" "this" {
  for_each = var.gcp_projects

  cloud            = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = each.value.project_id
  resource_suffix  = castai_external_connection_principals.this[each.key].resource_suffix

  dynamic "enabled_features" {
    for_each = each.value.features
    content {
      feature = enabled_features.value
    }
  }

  metadata {
    gcp {
      service_account_emails = each.value.service_account_emails
    }
  }
}

# -----------------------------------------------------------------------------
# Single explicit connection with sub-features and a pinned registry_version.
# This shows the fully expanded block form — useful for reviewing the raw
# interaction without dynamic blocks.
# -----------------------------------------------------------------------------

variable "gcp_detailed_project_id" {
  description = "GCP project for the explicit sub-feature demo connection."
  type        = string
  default     = "my-gcp-project-detailed"
}

resource "castai_external_connection_principals" "detailed" {
  cloud_provider   = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_detailed_project_id

  features {
    feature = "NODE_AUTOSCALING"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
    sub_features = [
      "NODE_AUTOSCALING_SPOT_INTERRUPTION_HANDLING",
      "NODE_AUTOSCALING_POD_PINNING",
    ]
  }

  features {
    feature = "WORKLOAD_AUTOSCALING"
    sub_features = [
      "WORKLOAD_AUTOSCALING_RESOURCE_QUOTA_AWARE_SCALING",
    ]
  }
}

resource "castai_external_connection" "detailed" {
  cloud            = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_detailed_project_id
  resource_suffix  = castai_external_connection_principals.detailed.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
    # registry_version = "v1.2.3" # uncomment to pin a specific permissions snapshot
    sub_features = [
      "NODE_AUTOSCALING_SPOT_INTERRUPTION_HANDLING",
      "NODE_AUTOSCALING_POD_PINNING",
    ]
  }

  enabled_features {
    feature = "WORKLOAD_AUTOSCALING"
    sub_features = [
      "WORKLOAD_AUTOSCALING_RESOURCE_QUOTA_AWARE_SCALING",
    ]
  }

  metadata {
    gcp {
      service_account_emails = {
        NODE_AUTOSCALING     = "node-autoscaling@my-gcp-project-detailed.iam.gserviceaccount.com"
        WORKLOAD_AUTOSCALING = "workload-autoscaling@my-gcp-project-detailed.iam.gserviceaccount.com"
      }
    }
  }
}

# -----------------------------------------------------------------------------
# (Optional) Org-level connection using the GCP_ORGANIZATION scope. The
# scope_key is the GCP organization ID instead of a project ID. Uncomment to try
# this interaction.
# -----------------------------------------------------------------------------

# variable "gcp_organization_id" {
#   description = "GCP organization ID for the org-scoped connection."
#   type        = string
#   default     = "123456789012"
# }

# resource "castai_external_connection_principals" "org" {
#   cloud_provider   = "GCP"
#   connection_scope = "GCP_ORGANIZATION"
#   scope_key        = var.gcp_organization_id
#
#   features {
#     feature = "COST_MONITORING"
#     sub_features = [
#       "COST_MONITORING_GPU_MONITORING",
#     ]
#   }
# }

# resource "castai_external_connection" "org" {
#   cloud            = "GCP"
#   connection_scope = "GCP_ORGANIZATION"
#   scope_key        = var.gcp_organization_id
#   resource_suffix  = castai_external_connection_principals.org.resource_suffix
#
#   enabled_features {
#     feature = "COST_MONITORING"
#     sub_features = [
#       "COST_MONITORING_GPU_MONITORING",
#     ]
#   }
#
#   metadata {
#     gcp {
#       service_account_emails = {
#         COST_MONITORING = "cost-monitoring@my-gcp-project-org.iam.gserviceaccount.com"
#       }
#     }
#   }
# }

# -----------------------------------------------------------------------------
# Outputs for inspection after apply.
# -----------------------------------------------------------------------------

output "connection_ids" {
  description = "IDs of the GCP external connections, keyed by project label."
  value = {
    for k, conn in castai_external_connection.this : k => conn.id
  }
}

output "detailed_connection_id" {
  description = "ID of the explicit sub-feature demo connection."
  value        = castai_external_connection.detailed.id
}

output "provisioned_principals" {
  description = "CAST-side provisioned resources per project, keyed by label."
  value = {
    for k, p in castai_external_connection_principals.this : k => p.provisioned_resources
  }
}
