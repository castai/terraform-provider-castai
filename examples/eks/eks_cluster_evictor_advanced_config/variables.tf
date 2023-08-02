# EKS module variables.
variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

variable "cluster_region" {
  type        = string
  description = "AWS Region in which EKS cluster and supporting resources will be created."
}

variable "cluster_version" {
  type        = string
  description = "EKS cluster version."
  default     = "1.24"
}

variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

# Variables required for connecting EKS cluster to CAST AI.
variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
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

variable "evictor_advanced_config" {
  type        = map(any)
  description = "Optional parameter, if set - advanced evictor configuration will be applied to cluster. For more information see https://docs.cast.ai/docs/evictor-configuration"
  default     = {
    "config.yaml" = <<EOT
evictionConfig:
  - podSelector:
      labelSelector:
        matchLabels:
          app.kubernetes.io/name: example
    settings:
      removalDisabled:
        enabled: true
EOT
  }
}


