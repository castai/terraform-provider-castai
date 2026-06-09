# GKE module variables.
variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
}

variable "cluster_region" {
  type        = string
  description = "The region to create the cluster."
}

variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster would be created."
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI api url"
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

variable "service_accounts_unique_ids" {
  type        = list(string)
  description = "Service Accounts' unique IDs used by node pools in the cluster."
  default     = []
}
