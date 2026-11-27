variable "castai_organization_id" {
  type        = string
  description = "CAST AI organization ID that owns the SSO connection."
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI api url."
  default     = "https://api.cast.ai"
}
