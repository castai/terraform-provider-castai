variable "castai_api_token" {
  type        = string
  description = "Name of the AKS cluster, resources will be created for."
}

variable "aks_cluster_name" {
  type        = string
  description = "Name of the AKS cluster, resources will be created for."
}

variable "aks_cluster_region" {
  type        = string
  description = "Region of the AKS cluster, resources will be created for."
}
