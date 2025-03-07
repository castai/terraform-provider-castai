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

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "grpc.cast.ai:443"
}

variable "kvisor_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "kvisor.prod-master.cast.ai:443"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID"
}

variable "tenant_id" {
  type        = string
  description = "Azure tenant ID"
}

variable "additional_resource_groups" {
  type    = list(string)
  default = []
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
