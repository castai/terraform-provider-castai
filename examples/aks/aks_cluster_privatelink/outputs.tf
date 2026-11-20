output "resource_group_name" {
  description = "Resource group holding the cluster, the private endpoint and the private DNS zone."
  value       = azurerm_resource_group.this.name
}

output "aks_cluster_name" {
  description = "Name of the AKS cluster."
  value       = azurerm_kubernetes_cluster.this.name
}

output "castai_cluster_id" {
  description = "CAST AI cluster ID of the onboarded cluster."
  value       = module.castai-aks-cluster.cluster_id
}

output "private_endpoint_ip" {
  description = "Private IP of the private endpoint NIC - what CAST AI hostnames resolve to inside the VNet."
  value       = azurerm_private_endpoint.castai.private_service_connection[0].private_ip_address
}

output "private_endpoint_id" {
  description = "Resource ID of the private endpoint, used to check the connection state."
  value       = azurerm_private_endpoint.castai.id
}

output "castai_private_dns_zone" {
  description = "Private DNS zone created for CAST AI hostnames."
  value       = azurerm_private_dns_zone.castai.name
}

output "castai_private_dns_fqdns" {
  description = "CAST AI hostnames that resolve to the private endpoint inside the VNet."
  value       = [for record in azurerm_private_dns_a_record.castai : "${record.name}.${azurerm_private_dns_zone.castai.name}"]
}

output "kubeconfig_command" {
  description = "Command that writes the kube-context for the cluster."
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.this.name} --name ${azurerm_kubernetes_cluster.this.name} --overwrite-existing"
}
