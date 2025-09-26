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
  default     = []
}

variable "project_id" {
  type        = string
  description = "GCP project ID in which GKE cluster would be created."
}

variable "castai_public_api_url" {
  type        = string
  description = "URL of public CAST AI API"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section."
}

variable "castai_api_private_domain" {
  type        = string
  description = "Private domain used to access Cast AI via Private Service Connect"
  default     = "prod-master.cast.ai"
}

variable "cast_api_service_attachment_uri" {
  type        = string
  description = "Service Attachment URI to connect to."
  default     = "projects/prod-master-scl0/regions/us-east4/serviceAttachments/castware-psc"
}

variable "allow_psc_global_access" {
  type        = bool
  description = "Allow global access to the Private Service Connect Endpoint. If set to false, the cluster must be in the same region as the Service Attachment."
  default     = true
}

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