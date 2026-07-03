# EKS module variables.
variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type        = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

# Variables required for connecting EKS cluster to CAST AI
variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
}

variable "kvisor_grpc_addr" {
  type        = string
  description = "CAST AI Kvisor gRPC address."
  default     = "kvisor.prod-master.cast.ai:443"
}
