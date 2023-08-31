variable "castai_dev_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section for development environment."
}

variable "castai_prod_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section for production environment."
}

variable "castai_dev_organization_name" {
  type        = string
  description = "Organization name used for development environment."
}

variable "castai_prod_organization_name" {
  type        = string
  description = "Organization name used for production environment."
}