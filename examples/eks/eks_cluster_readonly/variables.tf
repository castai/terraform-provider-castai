# EKS module variables.
variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

variable "cluster_version" {
  type        = string
  description = "EKS cluster version."
  default     = "1.23"
}

# Variables required for connecting EKS cluster to CAST AI
variable "castai_api_token" {
  type = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}
