variable "cluster_name" {
  type        = string
  description = "Name of the AKS cluster to be connected to the CAST AI."
}

variable "cluster_region" {
  type        = string
  description = "Region of the cluster to be connected to CAST AI."
}

variable "resource_group" {
  type        = string
  description = "Azure resource group that contains the cluster."
}

variable "subnets" {
  type        = list(string)
  description = "Subnet IDs used by CAST AI to provision nodes."
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID."
}

variable "additional_resource_groups" {
  type    = list(string)
  default = []
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optionally delete Cast AI created nodes when the cluster is destroyed."
  default     = true
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}
