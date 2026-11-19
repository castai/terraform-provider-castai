variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
}

variable "cluster_location" {
  type        = string
  description = "Location of the cluster to be connected to CAST AI. Can be region or zone for zonal clusters"
}

variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster would be created."
}

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
  default     = "grpc.cast.ai:443"
}

variable "kvisor_grpc_addr" {
  type        = string
  description = "Kvisor gRPC endpoint (host:port only). Must match the CAST AI console environment (default production: prod-master; EU: prod-eu)."
  default     = "kvisor.prod-master.cast.ai:443"
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
