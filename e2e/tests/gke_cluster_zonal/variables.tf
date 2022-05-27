variable "cluster_name" {
  type = string
}
variable "cluster_location" {
  type = string 
}

variable "network_region" {
  type = string
}

variable "project_id" {
  type = string 
}
variable "castai_api_token" {
  type = string 
}

variable "castai_api_url" {
  type = string
  default = "https://api.cast.ai"
}

variable "cluster_zones" {
  type        = list(string)
  default = ["europe-west1-b", "europe-west1-c"]
}

variable "gcp_credentials_base64" {
  type = string
}
