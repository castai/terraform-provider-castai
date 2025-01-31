variable "cluster_name" {
  type        = string
  description = "Name of the AKS cluster to be connected to the CAST AI."
}

variable "cluster_region" {
  type        = string
  description = "Region of the cluster to be connected to CAST AI."
}

variable "resource_group_name" {
  type        = string
  description = "Name of Azure Resource group that will be created for the cluster."
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

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "grpc.cast.ai:443"
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID"
}

variable "fqdn_without_proxy" {
  type        = list(string)
  description = "FQDNs that will be allowed on the AKS egress firewall and will not require proxy setup."
  default     = []
}