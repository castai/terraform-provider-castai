variable "cast_ai_api_key" {
  description = "Your CAST AI API key"
  type        = string
  sensitive   = true
  default     = "" # add your api key
}

variable "cluster_name" {
  description = "Name of your cluster"
  type        = string
  default     = "" # give a name for your cluster
}

variable "cluster_id" {
  description = "Identifier for your CAST AI cluster"
  type        = string
  default     = "" # add the cluster ID you copied from the UI from the 1st apply
}

variable "organization_id" {
  description = "Your CAST AI Organization ID"
  type        = string
  default     = "" # add your Org ID ..this is required for pod mutator deployment
}

