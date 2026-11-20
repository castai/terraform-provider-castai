# Azure and AKS variables.

variable "subscription_id" {
  type        = string
  description = "Azure subscription where the cluster, the private endpoint and the DNS zone are created. It must be on the CAST AI Private Link auto-approval list, otherwise the connection stays Pending."
}

variable "cluster_name" {
  type        = string
  description = "Name of the AKS cluster and of the resource group that holds every resource of this example. Must be unique in the subscription."
}

variable "cluster_region" {
  type        = string
  description = "Region of the cluster and of the private endpoint."
}

variable "aks_kubernetes_version" {
  type        = string
  description = "AKS Kubernetes version. Null (recommended) means the region default, so the example never pins a retired version."
  default     = null
}

variable "aks_vm_size" {
  type        = string
  description = "VM size of the system node pool. Two nodes are needed to run the CAST AI components comfortably."
  default     = "Standard_D2s_v5"
}

variable "aks_node_count" {
  type        = number
  description = "Node count of the system node pool."
  default     = 2
}

# CAST AI variables.

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token created in console.cast.ai -> API access keys."
  sensitive   = true
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI REST API URL."
  default     = "https://api.cast.ai"
}

variable "castai_grpc_url" {
  type        = string
  description = "CAST AI gRPC URL used by the pod pinner."
  default     = "grpc.cast.ai:443"
}

variable "castai_api_grpc_url" {
  type        = string
  description = "CAST AI gRPC API URL used by Kvisor and other components."
  default     = "api-grpc.cast.ai:443"
}

variable "castai_files_url" {
  type        = string
  description = "Static file proxies for CAST AI node binaries."
  default     = "files.cast.ai:443"
}

variable "castai_kvisor_grpc_addr" {
  type        = string
  description = "CAST AI Kvisor gRPC address."
  default     = "kvisor.prod-master.cast.ai:443"
}

variable "castai_telemetry_url" {
  type        = string
  description = "CAST AI telemetry endpoint used by the Kvisor reliability-metrics exporter."
  default     = "telemetry.prod-master.cast.ai:443"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Delete CAST AI provisioned nodes from Azure when the cluster is disconnected. Set to false for production clusters."
  default     = true
}

variable "tags" {
  type        = map(string)
  description = "Extra tags merged into the tags of every resource of this example."
  default     = {}
}

# Private Link variables.

variable "private_link_service_alias" {
  type        = string
  description = <<-EOT
    Alias of the CAST AI Private Link Service (producer side). Take it from CAST AI documentation.
  EOT

  validation {
    condition     = can(regex("^[A-Za-z0-9._-]+\\.azure\\.privatelinkservice$", var.private_link_service_alias))
    error_message = "private_link_service_alias must be a Private Link Service alias ending in .azure.privatelinkservice (the alias, not the resource ID)."
  }
}

variable "castai_private_dns_zone" {
  type        = string
  description = <<-EOT
    Name of the private DNS zone that steers CAST AI hostnames to the private
    endpoint. In most deployments, keep it at "cast.ai": the safest approach to cover all hostnames automatically.
  EOT
  default     = "cast.ai"
}
