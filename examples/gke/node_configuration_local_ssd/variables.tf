variable "castai_cluster_id" {
  type        = string
  description = "CAST AI cluster id (uuid) of the onboarded GKE cluster. Required when onboard_cluster = false."
}

variable "onboard_cluster" {
  type        = bool
  default     = false
  description = "When true, registers the GKE cluster (below) with CAST AI first and creates the node configuration on it. Set to false to reuse an already onboarded cluster via castai_cluster_id."
}

variable "gke_cluster_name" {
  type        = string
  default     = ""
  description = "GKE cluster name (onboarding path only)."
}

variable "gke_project_id" {
  type        = string
  default     = ""
  description = "GCP project id (onboarding path only)."
}

variable "gke_cluster_location" {
  type        = string
  default     = ""
  description = "GKE cluster zone (zonal) or region (regional) (onboarding path only)."
}

variable "gke_credentials_json" {
  type        = string
  default     = ""
  sensitive   = true
  description = "GCP service account credentials.json for CAST AI to access the cluster. Either this or the impersonation pair (cast_service_account_email + target_service_account_email) is required on the onboarding path."
}

variable "cast_service_account_email" {
  type        = string
  default     = ""
  description = "CAST AI per-org impersonation (jumper) service account email - get it from the castai_impersonation_service_account data source with provider = \"gcp\". Set together with target_service_account_email instead of gke_credentials_json."
}

variable "target_service_account_email" {
  type        = string
  default     = ""
  description = "Customer-side GCP service account for CAST AI to impersonate. Must have the CAST AI GKE roles (see the castai/gke-iam module) and must allow token creation by the jumper service account. Set together with cast_service_account_email."
}

variable "subnets" {
  type        = list(string)
  description = "Subnetwork self-links for the node configuration; GKE requires at least one. Find it with:\n  gcloud container clusters describe <cluster> --region <region> --format='value(subnetwork)'"
}

variable "local_ssd_mode" {
  type        = string
  default     = "MIXED"
  description = "How local SSDs are exposed on the node: EPHEMERAL, MIXED or BLOCK. MIXED requires ephemeral_storage_percent."

  validation {
    condition     = contains(["EPHEMERAL", "MIXED", "BLOCK"], var.local_ssd_mode)
    error_message = "local_ssd_mode must be one of EPHEMERAL, MIXED, BLOCK."
  }
}

variable "min_local_storage_gib" {
  type        = number
  default     = 375
  description = "Minimum local SSD capacity (GiB). Set to 0 to omit it from the request."
}

variable "local_storage_per_cpu_gib" {
  type        = number
  default     = 3.5
  description = "Additional local SSD capacity per vCPU (GiB). Set to 0 to omit it from the request."
}

variable "ephemeral_storage_percent" {
  type        = number
  default     = 30
  description = "MIXED only: percentage (1-99) of the total local SSD capacity striped into kubelet ephemeral storage. Set to 0 to omit it from the request."
}

variable "min_iops" {
  type        = number
  default     = 1000
  description = "Minimum IOPS for the local SSD storage. Valid for all modes. Set to 0 to omit it from the request."
}
