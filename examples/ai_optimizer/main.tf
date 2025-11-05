terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
  }
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI API URL"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token"
  sensitive   = true
}

variable "organization_id" {
  type        = string
  description = "CAST AI organization ID"
}

variable "cluster_id" {
  type        = string
  description = "CAST AI cluster ID"
}

# Create an AI Optimizer API Key
resource "castai_ai_optimizer_api_key" "example" {
  organization_id = var.organization_id
  name            = "example-ai-optimizer-key"
}

# Output the generated API key token (sensitive)
output "ai_optimizer_api_key_token" {
  value     = castai_ai_optimizer_api_key.example.token
  sensitive = true
}

# Retrieve hosted models for a cluster
data "castai_ai_optimizer_hosted_models" "example" {
  organization_id = var.organization_id
  cluster_id      = var.cluster_id
}

# Output the list of hosted models
output "hosted_models" {
  value = data.castai_ai_optimizer_hosted_models.example.models
}
