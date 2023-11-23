variable "castai_api_url" {
  type    = string
  default = "https://api.cast.ai"
}

variable "castai_api_token" {
  type      = string
  sensitive = true
}

variable "gcp_credentials" {
  type        = string
  sensitive   = true
  description = "Credentials in base64 format"
}

variable "service_account_id" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "cluster_region" {
  type = string
}

variable "cluster_zones" {
  type = list(string)
}

variable "project_id" {
  type = string
}
