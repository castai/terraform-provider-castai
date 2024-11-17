# EKS module variables.
variable "profile" {
  type        = string
  description = "Profile used with AWS CLI"
  default     = "default"
}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type        = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

variable "vpc_id" {
  type        = string
  description = "EKS cluster VPC ID"
}

variable "castai_api_url" {
  type        = string
  description = "URL of CAST AI API"
  default     = "https://api.cast.ai"
}

# Variables required for connecting EKS cluster to CAST AI.
variable "castai_api_key" {
  type        = string
  description = "CAST AI API Key created in console.cast.ai API Access keys section"
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

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}


variable "cluster_security_group_id" {
  type        = string
  description = "EKS cluster security group ID"
}

variable "node_security_group_id" {
  type        = string
  description = "EKS cluster node security group ID"
}

variable "subnets" {
  type        = list(string)
  description = "Subnet IDs used by CAST AI to provision nodes"
}
