# AKS  cluster variables.
variable "cluster_name" {
  type        = string
  description = "Name of the AKS cluster, resources will be created for."
}

variable "cluster_region" {
  type        = string
  description = "Region of the AKS cluster, resources will be created for."
}

variable "castai_api_url" {
  type = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

# Variables required for connecting EKS cluster to CAST AI
variable "castai_api_token" {
  type = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection. For production use it is recommended to set it to false."
  default     = true
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}

variable "cluster_resource_group_name" {
  type = string
  description = "Name of resource group in which cluster was created"
}

variable "subnet_name" {
  type = string
  description = "Name of subnet used for provisioning CAST AI nodes"
}

variable "vnet_name" {
  type = string
  description = "Name of virtual network used for provisioning CAST AI nodes"
}

variable "subnet_resource_group_name" {
  type = string
  description = "Name of resource group in which vnet was created"
}