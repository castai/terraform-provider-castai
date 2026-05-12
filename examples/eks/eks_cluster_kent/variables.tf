variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type        = string
  description = "AWS region in which the EKS cluster and supporting resources will be created."
}

variable "cluster_version" {
  type        = string
  description = "EKS cluster version."
  default     = "1.34"
}

variable "tags" {
  type        = map(string)
  description = "Tags applied to created AWS resources."
  default     = {}
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai → Settings → API access. Sensitive."
  sensitive   = true
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI API URL."
  default     = "https://api.cast.ai"
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL (used by castai-agent and castai-kentroller for the StreamActions connection)."
  default     = "grpc.cast.ai:443"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "If true, CAST AI-provisioned nodes are deleted from the cloud on cluster disconnect. For production use, recommended to set to false."
  default     = true
}
