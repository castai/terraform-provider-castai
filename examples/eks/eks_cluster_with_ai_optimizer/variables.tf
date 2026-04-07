# EKS Cluster variables.
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
  default     = "1.33"
}

# CAST AI variables.
variable "castai_api_url" {
  type        = string
  description = "URL of alternative CAST AI API to be used during development or testing"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "castai_organization_id" {
  type        = string
  description = "CAST AI organization ID. Required when the API token has access to multiple organizations."
  default     = ""
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL"
  default     = "grpc.cast.ai:443"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from cloud on cluster disconnection."
  default     = true
}

# AI Optimizer variables.
variable "enable_ai_optimizer" {
  type        = bool
  description = "Enable CAST AI AI Optimizer for LLM model serving."
  default     = true
}

variable "model_registry_bucket" {
  type        = string
  description = "S3 bucket name for private model registry. Required when deploying custom models."
  default     = ""
}

variable "model_registry_region" {
  type        = string
  description = "AWS region for the model registry S3 bucket."
  default     = "us-east-1"
}

variable "hf_token" {
  type        = string
  description = "Hugging Face token for accessing Hugging Face Hub models. Required when deploying predefined models with vLLM configuration."
}

variable "deploy_predefined_model" {
  type        = bool
  description = "Deploy a predefined CastAI-managed model using an existing model specs ID."
  default     = true
}

variable "deploy_custom_model" {
  type        = bool
  description = "Deploy a custom model from S3 registry."
  default     = false
}

variable "custom_model_name" {
  type        = string
  description = "Name for the custom model."
  default     = "my-custom-model"
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}
