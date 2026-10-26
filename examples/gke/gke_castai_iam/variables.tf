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