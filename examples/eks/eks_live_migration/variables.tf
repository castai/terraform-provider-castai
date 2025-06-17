variable "cluster_name" {
  type        = string
  description = "Name of the cluster to create"
}

variable "region" {
  description = "AWS region where cluster will be created"
  type        = string
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI api token"
  sensitive   = true
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

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection. For production use it is recommended to set it to false."
  default     = true
}