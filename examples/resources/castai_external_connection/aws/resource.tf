# Example: AWS external connection enabling node autoscaling, workload autoscaling (woop),
# and cost monitoring. This configuration is for reference only — do not apply without
# reviewing the placeholder values below.
#
# Flow:
#   1. castai_external_connection_principals provisions CAST-side IAM principals and
#      emits a resource_suffix output.
#   2. castai_external_connection creates the connection, passing that resource_suffix
#      back to CAST AI along with the customer-side IAM role ARN.

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
  description = "AWS account ID to connect to CAST AI."
  type        = string
  default     = "123456789012"
}

variable "aws_role_arn" {
  description = "IAM role ARN in the customer's AWS account that CAST AI assumes."
  type        = string
  default     = "arn:aws:iam::123456789012:role/castai-external-connection"
}

# Provision CAST-side IAM principals (roles, policies) for the requested features.
# The resource_suffix output is passed to the castai_external_connection resource.
resource "castai_external_connection_principals" "this" {
  cloud_provider   = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_account_id

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

# Create the external connection linking the AWS account to CAST AI.
# The resource_suffix from the principals resource establishes the trust relationship.
resource "castai_external_connection" "this" {
  cloud            = "AWS"
  connection_scope = "AWS_ACCOUNT"
  scope_key        = var.aws_account_id
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
    aws {
      role_arn = var.aws_role_arn
    }
  }
}
