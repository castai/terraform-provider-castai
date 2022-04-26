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

