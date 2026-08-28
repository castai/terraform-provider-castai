# Example: GCP external connection enabling node autoscaling, workload autoscaling (woop),
# and cost monitoring. This configuration is for reference only — do not apply without
# reviewing the placeholder values below.
#
# Flow:
#   1. castai_external_connection_principals provisions CAST-side IAM principals and
#      emits a resource_suffix output.
#   2. castai_external_connection creates the connection, passing that resource_suffix
#      back to CAST AI along with the customer-side GCP service account emails.

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

variable "gcp_project_id" {
  description = "GCP project ID to connect to CAST AI."
  type        = string
  default     = "my-gcp-project-123"
}

variable "gcp_service_account_emails" {
  description = "Map of feature ID to the customer-side GCP service account email created in the customer's project. Keys must match the feature enum values."
  type        = map(string)
  default = {
    NODE_AUTOSCALING     = "node-autoscaling@my-gcp-project-123.iam.gserviceaccount.com"
    WORKLOAD_AUTOSCALING = "workload-autoscaling@my-gcp-project-123.iam.gserviceaccount.com"
    COST_MONITORING      = "cost-monitoring@my-gcp-project-123.iam.gserviceaccount.com"
  }
}

# Provision CAST-side IAM principals (service accounts, roles) for the requested features.
# The resource_suffix output is passed to the castai_external_connection resource.
resource "castai_external_connection_principals" "this" {
  cloud_provider   = "GCP"
  scope_key        = var.gcp_project_id

  connection_scope = "GCP_PROJECT"
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

# Create the external connection linking the GCP project to CAST AI.
# The resource_suffix from the principals resource establishes the trust relationship.
resource "castai_external_connection" "this" {
  cloud            = "GCP"
  connection_scope = "GCP_PROJECT"
  scope_key        = var.gcp_project_id
  resource_suffix  = castai_external_connection_principals.this.resource_suffix

  enabled_features {
    feature = "NODE_AUTOSCALING"
    # resource_ids = ["some-ID-of-the-cluster"];
  }

  enabled_features {
    feature = "WORKLOAD_AUTOSCALING"
  }

  enabled_features {
    feature = "COST_MONITORING"
  }

  metadata {
    gcp {
      service_account_emails = var.gcp_service_account_emails
    }
  }
}
