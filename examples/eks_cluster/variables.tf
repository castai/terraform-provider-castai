variable "castai_api_token" {
  default = ""
}

variable "aws_account_id" {
  default = ""
}

variable "aws_access_key_id" {
  default = ""
}

variable "aws_secret_access_key" {
  default = ""
}

variable "cluster_region" {
  type        = string
  description = "AWS region your cluster is located."
  default = "eu-central-1"
}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
  default = ""
}


# Addresses with whitelisted access to Kubernetes API
variable "whitelisted_ips" {
  type = list(string)
  default = ["0.0.0.0/0"]
}

variable "subnets" {
  type = list(string)
  description = "Optional custom subnets for the cluster. If not set subnets from the EKS cluster configuration are used."
  default = []
}

variable "security_groups" {
  type = list(string)
  description = "Optional custom security groups for the cluster. If not set security groups from the EKS cluster configuration are used."
  default = []
}

variable "tags" {
  type = map
  description = "Optional tags for new cluster nodes. This parameter applies only to new nodes - tags for old nodes are not reconciled."
  default = {}
}

variable "delete_nodes_on_disconnect" {
  type = bool
  description = "Optional parameter, if set to true - CAST AI provisioned nodes will be deleted from EC2 on cluster disconnection."
  default = false
}
