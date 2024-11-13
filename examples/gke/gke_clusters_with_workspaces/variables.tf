variable "gke_project_id" {
  type        = string
  description = "The project id from GCP"
  default     = ""
}

variable "gke_cluster_name" {
  type        = string
  description = "Name of the cluster to be connected to CAST AI."
  default     = ""
}

variable "gke_cluster_location" {
  type        = string
  description = "Location of the cluster to be connected to CAST AI. Can be region or zone for zonal clusters"
  default     = ""
}

variable "gke_subnets" {
  type        = list(string)
  description = "Subnet IDs used by CAST AI to provision nodes."
  default     = []
}

variable "service_accounts_unique_ids" {
  type        = list(string)
  description = "Service Accounts' unique IDs used by node pools in the cluster."
  default     = []
}

variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "Optional CAST AI API token created in console.cast.ai API Access keys section. Used only when `wait_for_cluster_ready` is set to true"
  sensitive   = true
  default     = ""
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optionally delete Cast AI created nodes when the cluster is destroyed"
  default     = false
}

variable "castai_components_labels" {
  type        = map(any)
  description = "Optional additional Kubernetes labels for CAST AI pods"
  default     = {}
}

variable "agent_version" {
  description = "Version of castai-agent helm chart. Default latest"
  type        = string
  default     = null
}

variable "cluster_controller_version" {
  description = "Version of castai-cluster-controller helm chart. Default latest"
  type        = string
  default     = null
}

variable "evictor_version" {
  description = "Version of castai-evictor chart. Default latest"
  type        = string
  default     = null
}

variable "agent_values" {
  description = "List of YAML formatted string values for agent helm chart"
  type        = list(string)
  default     = []
}

variable "cluster_controller_values" {
  description = "List of YAML formatted string values for cluster-controller helm chart"
  type        = list(string)
  default     = []
}

variable "evictor_values" {
  description = "List of YAML formatted string values for evictor helm chart"
  type        = list(string)
  default     = []
}

variable "kube_config_context" {
  type        = string
  description = "kube_config_context"
  default     = ""
}

variable "org_workspace" {
  type        = string
  description = "organization terraform workspace"
  default     = "org-workspace"
}
