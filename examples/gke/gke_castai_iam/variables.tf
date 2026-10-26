variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster is located."
}

variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
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

variable "service_accounts_unique_ids" {
  type        = list(string)
  description = "Service Accounts' unique IDs used by node pools in the cluster."
  default     = []
}

variable "authentication_method" {
  type        = string
  description = "Authentication method for CAST AI. Use 'service_account_key' for a service account with a JSON private key, or 'impersonation' for GKE service account impersonation."
  default     = "impersonation"

  validation {
    condition     = contains(["service_account_key", "impersonation"], var.authentication_method)
    error_message = "authentication_method must be either 'service_account_key' or 'impersonation'."
  }
}
