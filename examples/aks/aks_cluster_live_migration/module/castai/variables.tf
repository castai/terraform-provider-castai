variable "cluster_name" {
  type        = string
  description = "AKS cluster name."
}

variable "cluster_region" {
  type        = string
  description = "AKS cluster region."
}

variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "grpc.cast.ai:443"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection."
  default     = true
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes."
  default     = {}
}

variable "node_resource_group" {
  type        = string
  description = "AKS node resource group name."
}

variable "resource_group" {
  type        = string
  description = "AKS cluster resource group name."
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID."
}

variable "tenant_id" {
  type        = string
  description = "Azure tenant ID."
}

variable "subnet_id" {
  type        = string
  description = "Azure subnet ID for CAST AI nodes."
}

variable "install_helm_live" {
  type        = bool
  description = "If set to true - the 'castai-live' Helm chart will be installed on the cluster. Required for live migration functionality."
  default     = true
}

variable "live_helm_version" {
  type        = string
  description = "Version of the castai-live Helm chart to install."
  default     = "0.91.0"
}
