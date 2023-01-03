# GKE module variables.
variable "cluster_name" {
  type        = string
  description = "GKE cluster name in GCP project."
}

variable "cluster_region" {
  type        = string
  description = "The region to create the cluster."
}

variable "cluster_zones" {
  type        = list(string)
  description = "The zones to create the cluster."

  default = ["europe-west1-b", "europe-west1-c"]
}

variable "project_id" {
  type = string
  description = "GCP project ID in which GKE cluster would be created."
}

# Variables required for connecting EKS cluster to CAST AI
variable "castai_api_token" {
  type = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection."
  default     = true
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}
