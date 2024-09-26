## Required variables.

variable "aws_account_id" {
  type        = string
  description = "ID of AWS account the cluster is located in."
}

variable "aws_cluster_region" {
  type        = string
  description = "Region of the cluster to be connected to CAST AI."
}

variable "aws_cluster_name" {
  type        = string
  description = "Name of the cluster to be connected to CAST AI."
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "aws_assume_role_arn" {
  type        = string
  description = "Arn of the role to be used by CAST AI for IAM access"
  default     = null
}

variable "subnets" {
  type        = list(string)
  description = "Subnet IDs used by CAST AI to provision nodes"
}

variable "cluster_security_group_id" {
  type        = string
  description = "EKS cluster security group ID"
}

variable "node_security_group_id" {
  type        = string
  description = "EKS cluster node security group ID"
}

variable "vpc_id" {
  type        = string
  description = "EKS cluster VPC ID"
}

variable "profile" {
  type        = string
  description = "Profile used with AWS CLI"
  default     = "default"
}

## Optional variables.

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optionally delete Cast AI created nodes when the cluster is destroyed"
  default     = false
}
