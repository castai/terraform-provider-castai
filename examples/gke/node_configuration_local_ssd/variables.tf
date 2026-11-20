variable "castai_cluster_id" {
  type        = string
  description = "CAST AI cluster id (uuid) of the onboarded GKE cluster."
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
