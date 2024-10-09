# GKE module variables.
variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
}

variable "cluster_region" {
  type        = string
  description = "The region to create the cluster."
}

variable "cluster_read_only" {
  type        = bool
  description = "Whether cluster should be read-only."
}

variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster would be created."
}

# Variables required for connecting EKS cluster to CAST AI
variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "api-grpc.cast.ai:443"
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}

