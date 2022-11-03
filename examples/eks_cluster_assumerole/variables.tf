variable "castai_api_token" {
  type        = string
  description = "CAST.AI api token"
  default     = ""
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI API url"
  default     = "https://api.cast.ai"
}

variable "aws_access_key_id" {
  type        = string
  description = "Your own access key id for operating terraform"
  default     = ""
}

variable "aws_secret_access_key" {
  type        = string
  description = "Your own access key secret for operating terraform"
  default     = ""
}

variable "cluster_region" {
  type        = string
  description = "AWS region your cluster is located."
  default     = ""
}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
  default     = ""
}

variable "tags" {
  type        = map(any)
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default     = {}
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from EC2 on cluster disconnection."
  default     = false
}
