# EKS module variables.
variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_version" {
  type        = string
  description = "EKS cluster name version."
  default     = "1.23"
}

variable "cluster_region" {
  type = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

# Variables required for connecting EKS cluster to CAST AI.
variable "castai_api_token" {
  type = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from EC2 on cluster disconnection."
  default     = true
}