variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
  default     = "gke-907-av"
}

variable "cluster_region" {
  type        = string
  description = "The region to create the cluster."
  default     = "us-central1"
}

variable "cluster_zones" {
  type        = list(string)
  description = "The zones to create the cluster."
  default     = ["us-central1-c"]
}

variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster would be created."
  default     = "demos-321800"
}

variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

# Variables required for connecting EKS cluster to CAST AI

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

variable "subnets" {
  type        = list(string)
  description = "Cluster subnets"
  default     = ["projects/demos-321800/regions/us-central1/subnetworks/gke-907-av-ip-range-nodes"]
}