# Example: GCP external connections.
#
# This configuration is for reference only — do not apply without reviewing the
# placeholder values below. It demonstrates the main interaction patterns of the
# castai_external_connection / castai_external_connection_principals resources:
#
#   1. A "full" connection on the primary GCP project: NODE_AUTOSCALING,
#      WORKLOAD_AUTOSCALING, and COST_MONITORING, each with its own
#      customer-side service account email.
#   2. A second connection on a *different* GCP project (new scope_key) with a
#      smaller feature set.
#   3. A third connection with sub-feature selection and an optional pinned
#      registry_version.
#   4. (Commented) an org-level connection using the GCP_ORGANIZATION scope.
#
# Everything is written out explicitly on purpose — duplication is fine here,
# the goal is to show exactly what the Terraform interaction looks like.
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

# -----------------------------------------------------------------------------
# Connection 1: primary GCP project, full feature set.
# -----------------------------------------------------------------------------

variable "gcp_primary_project_id" {
  description = "Primary GCP project ID to connect to CAST AI."
  type        = string
  default     = "my-gcp-project-prod"
}

# CAST-side IAM principals for the primary project. The resource_suffix output
# is passed to the castai_external_connection resource.
resource "castai_external_connection_principals" "primary" {
  cloud_provider   = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_primary_project_id

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

# Connection linking the primary GCP project to CAST AI. The resource_suffix
# from the principals resource establishes the trust relationship.
resource "castai_external_connection" "primary" {
  cloud            = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_primary_project_id
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
    gcp {
      service_account_emails = {
        NODE_AUTOSCALING     = "node-autoscaling@my-gcp-project-prod.iam.gserviceaccount.com"
        WORKLOAD_AUTOSCALING = "workload-autoscaling@my-gcp-project-prod.iam.gserviceaccount.com"
        COST_MONITORING      = "cost-monitoring@my-gcp-project-prod.iam.gserviceaccount.com"
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Connection 2: secondary GCP project (new scope_key), smaller feature set.
# -----------------------------------------------------------------------------

variable "gcp_secondary_project_id" {
  description = "Second GCP project connected to CAST AI (new scope_key demo)."
  type        = string
  default     = "my-gcp-project-nonprod"
}

resource "castai_external_connection_principals" "secondary_project" {
  cloud_provider   = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_secondary_project_id

  features {
    feature = "NODE_AUTOSCALING"
  }

  features {
    feature = "KARPENTER_ENTERPRISE"
  }
}

resource "castai_external_connection" "secondary_project" {
  cloud            = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_secondary_project_id
  resource_suffix  = castai_external_connection_principals.secondary_project.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
  }

  enabled_features {
    feature = "KARPENTER_ENTERPRISE"
  }

  metadata {
    gcp {
      service_account_emails = {
        NODE_AUTOSCALING     = "node-autoscaling@my-gcp-project-nonprod.iam.gserviceaccount.com"
        KARPENTER_ENTERPRISE = "karpenter@my-gcp-project-nonprod.iam.gserviceaccount.com"
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Connection 3: sub-feature selection and an optional pinned registry_version.
# -----------------------------------------------------------------------------

variable "gcp_detailed_project_id" {
  description = "GCP project for the sub-feature demo connection."
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

output "primary_connection_id" {
  description = "ID of the primary GCP external connection."
  value        = castai_external_connection.primary.id
}

output "secondary_project_connection_id" {
  description = "ID of the secondary GCP project external connection."
  value        = castai_external_connection.secondary_project.id
}

output "detailed_connection_id" {
  description = "ID of the sub-feature demo connection."
  value        = castai_external_connection.detailed.id
}

output "provisioned_principals" {
  description = "CAST-side provisioned resources for the primary connection."
  value        = castai_external_connection_principals.primary.provisioned_resources
}
