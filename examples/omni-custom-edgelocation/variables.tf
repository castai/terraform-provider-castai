variable "castai_api_token" {
  description = "Your CAST AI API token"
  type        = string
  sensitive   = true
}

variable "organization_id" {
  description = "Your CAST AI Organization ID"
  type        = string
}

variable "cluster_id" {
  description = "CAST AI Omni cluster ID"
  type        = string
}

variable "region" {
  description = "Region where the edge location is deployed. Can be omitted for custom cloud provider."
  type        = string
  default     = null
}

variable "edge_location_name" {
  description = "Name of the edge location. Must be unique and relatively short (max 30 chars for GCP service account compatibility)."
  type        = string
}

variable "description" {
  description = "Description of the edge location"
  type        = string
  default     = "Custom edge location onboarded by Terraform"
}

variable "tunneled_cidrs" {
  description = "List of destination CIDR blocks whose traffic should be routed through the main cluster."
  type        = list(string)
  default     = []
}

variable "cni_overlay" {
  description = "Overlay mode for kube-router pod-to-pod traffic. Valid: OVERLAY_UNSPECIFIED, OVERLAY_OFF, OVERLAY_SUBNET, OVERLAY_FULL."
  type        = string
  default     = "OVERLAY_FULL"

  validation {
    condition     = contains(["OVERLAY_UNSPECIFIED", "OVERLAY_OFF", "OVERLAY_SUBNET", "OVERLAY_FULL"], var.cni_overlay)
    error_message = "cni_overlay must be OVERLAY_UNSPECIFIED, OVERLAY_OFF, OVERLAY_SUBNET, or OVERLAY_FULL."
  }
}

variable "cni_overlay_encap" {
  description = "Encapsulation protocol used by the overlay. Valid: OVERLAY_ENCAP_UNSPECIFIED, OVERLAY_ENCAP_IPIP, OVERLAY_ENCAP_FOU."
  type        = string
  default     = "OVERLAY_ENCAP_FOU"

  validation {
    condition     = contains(["OVERLAY_ENCAP_UNSPECIFIED", "OVERLAY_ENCAP_IPIP", "OVERLAY_ENCAP_FOU"], var.cni_overlay_encap)
    error_message = "cni_overlay_encap must be OVERLAY_ENCAP_UNSPECIFIED, OVERLAY_ENCAP_IPIP, or OVERLAY_ENCAP_FOU."
  }
}
