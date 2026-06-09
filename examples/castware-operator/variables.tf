variable "castai_api_url" {
  type        = string
  description = "CAST AI API URL"
  default     = "https://api.cast.ai" # make this https://api.eu.cast.ai if you are targeting EU console
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token from https://console.cast.ai"
  sensitive   = true
}

variable "aks_cluster_name" {
  type        = string
  description = "Name of your existing AKS cluster"
}

variable "aks_cluster_region" {
  type        = string
  description = "Azure region (e.g., eastus, westeurope)"
}

#castware
variable "castware_operator_version" {
  type        = string
  description = "Castware operator version"
  default     = "0.0.25" # >= version that supports TF 
}

variable "extended_permissions" {
  type        = bool
  description = "Enable extended permissions to install phase2 components"
  default     = false # Set it to true to install cluster controller
}

variable "castware_operator_image_tag" {
  type        = string
  description = "Image tag for castware operator"
  default     = "0.0.25"
}

variable "castware_default_components_enabled" {
  type        = bool
  description = "Enable default components"
  default     = false # should ALWAYS be false for terraform
}


