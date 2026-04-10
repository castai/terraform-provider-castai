output "eks_cluster_authentication_mode" {
  value = data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode
}

output "cluster_id" {
  description = "CAST AI cluster ID"
  value       = castai_eks_clusterid.cluster_id.id
}

output "cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = data.aws_eks_cluster.existing_cluster.endpoint
}

output "cluster_name" {
  description = "EKS cluster name"
  value       = data.aws_eks_cluster.existing_cluster.name
}

output "predefined_model_service" {
  description = "Kubernetes service name for the predefined Llama 3.2 1B model"
  value       = var.deploy_predefined_model ? "llama3-2-1b" : null
}

output "custom_model_service" {
  description = "Kubernetes service name for the custom model"
  value       = var.deploy_custom_model ? "${var.custom_model_name}-service" : null
}

output "model_registry_id" {
  description = "ID of the model registry (if custom model is deployed)"
  value       = var.deploy_custom_model ? castai_ai_optimizer_model_registry.custom_models[0].id : null
}
