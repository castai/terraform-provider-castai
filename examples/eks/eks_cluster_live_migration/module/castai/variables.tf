variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type = string
}

variable "live_proxy_version" {
  type    = string
  default = "0.30.0"
}

variable "live_helm_version" {
  type    = string
  default = "0.1.43"
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}

variable "subnets" {}

variable "security_groups" {}

variable "vpc_id" {}

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
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection. For production use it is recommended to set it to false."
  default     = true
}

variable "install_helm_live" {
  type        = bool
  description = "Optional parameter, if set to true - the 'castai-live' Helm chart will be installed on the cluster. This is required for live migration feature."
  default     = true
}
