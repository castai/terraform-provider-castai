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
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL used by pod pinner"
  default     = "grpc.cast.ai:443"
}

variable "kvisor_grpc_addr" {
  type        = string
  description = "CAST AI Kvisor optimized GRPC API address"
  default     = "kvisor.prod-master.cast.ai:443" // If your cluster is in the EU region, update the grpcAddr to: kvisor.prod-eu.cast.ai:443
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

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID"
}
