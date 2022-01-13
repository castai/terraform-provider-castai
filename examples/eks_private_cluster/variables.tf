variable "castai_api_token" {
  type        = string
  description = "CAST.AI api token"
}

variable "aws_account_id" {
  type        = string
  description = "AWS account your cluster is located."
}

variable "aws_access_key_id" {
  type        = string
  description = "Your own access key id for operating terraform"
}

variable "aws_secret_access_key" {
  type        = string
  description = "Your own access key secret for operating terraform"
}

variable "cluster_region" {
  type        = string
  description = "AWS region your cluster is located."

}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name in AWS account."
}

# Addresses with whitelisted access to Kubernetes API
variable "whitelisted_ips" {
  type = list(string)
  default = []
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
