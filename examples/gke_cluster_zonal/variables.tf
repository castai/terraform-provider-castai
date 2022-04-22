variable "castai_api_token" {}

variable "project_id" {}
variable "cluster_region" {
  type = string
  description = "The region to create the cluster"

  default = "europe-west1"
}

variable "cluster_name" {}


variable "cluster_zones" {
  type        = list(string)
  description = "The zones to create the cluster."

  default = ["europe-west1-b", "europe-west1-c"]
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection."
  default     = false
}
