variable "castai_api_token" {
  type        = string
  description = "Name of the AKS cluster, resources will be created for."
}

variable "aks_cluster_name" {
  type        = string
  description = "Name of the AKS cluster, resources will be created for."
}

variable "aks_resource_group" {
  type        = string
  description = "Name of the AKS resource group that will be created for AKS cluster."
}
