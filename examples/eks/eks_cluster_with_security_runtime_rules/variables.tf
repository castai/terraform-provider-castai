# EKS module variables.
variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type        = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

variable "cluster_version" {
  type        = string
  description = "EKS cluster version."
  default     = "1.28"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
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

# EKS module variables.
variable "profile" {
  type        = string
  description = "Profile used with AWS CLI"
  default     = "default"
}