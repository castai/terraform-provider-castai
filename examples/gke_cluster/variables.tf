variable "castai_api_token" {}

variable "project_id" {}
variable "cluster_region" {}
variable "cluster_name" {}


variable "cluster_zones" {
  type        = list(string)
  description = "The zones to create the cluster."

  default = ["europe-west-1-a", "europe-west-1-b"]
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection."
  default     = false
}
