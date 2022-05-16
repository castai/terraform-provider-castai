variable "castai_api_token" {
  type        = string
  description = "CAST.AI api token"
}

variable "cluster_region" {
  type        = string
  description = "AWS region your cluster is located."
}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
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
