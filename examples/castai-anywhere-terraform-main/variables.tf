variable "cast_ai_api_key" {
  description = "Your CAST AI API key"
  type        = string
  sensitive   = true
  default     = "2793d7538951b4bfd8c6a4e5cff67675283ecfc5de7fbacd3c4a7b25b7aae2df" # add your api key
}


variable "cluster_name" {
  description = "Name of your cluster"
  type        = string
  default     = "minikube-test" # give a name for your cluster
}

variable "cluster_id" {
  description = "Identifier for your CAST AI cluster"
  type        = string
  default     = "89cfa5f2-ddcf-4cc7-9ada-57fa397ca56a" # add the cluster ID you copied from the UI from the 1st apply
}

variable "organization_id" {
  description = "Your CAST AI Organization ID"
  type        = string
  default     = "ca8670b8-3d78-47ab-be57-5024796b527f" # add your Org ID ..this is required for pod mutator deployment
}

variable "managed_by_castai" {
  description = "Flag to indicate if the components are managed by CAST AI"
  type        = bool
  default     = false # if its true, CAST overrides every changes made 
}
