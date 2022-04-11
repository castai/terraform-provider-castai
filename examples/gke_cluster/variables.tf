variable "castai_api_token" {}

variable "project_id" {}
variable "cluster_region" {}
variable "cluster_name" {}

variable "castai_api_url" {
  description = "CAST.AI API URL"
  default     = "https://api.cast.ai/"
}

variable "network_name" {
  description = "The VPC network created to host the cluster in"
  default     = "gke-network"
}

variable "ip_range_nodes_name" {
  description = "The ip range name to use for nodes"
  default     = "ip-range-nodes"
}

variable "ip_range_nodes_cidr" {
  description = "The ip range CIDR to use for nodes"

  default = "10.10.0.0/16"
}

variable "ip_range_pods_name" {
  description = "The ip range name to use for pods"
  default     = "ip-range-pods"
}

variable "ip_range_pods_cidr" {
  description = "The ip range CIDR to use for pods"

  default = "10.20.0.0/16"
}

variable "ip_range_services_name" {
  description = "The ip range name to use for services"
  default     = "ip-range-services"
}

variable "ip_range_services_cidr" {
  description = "The ip range CIDR to use for services"

  default = "10.30.0.0/24"
}

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
