output "cluster_id" {
  description = "CAST AI cluster ID"
  value       = castai_eks_clusterid.cluster_id.id
}

output "cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "predicted_model_service" {
  description = "Kubernetes service name for the predefined Llama 3.1 model"
  value       = var.deploy_predefined_model ? "llama31-service" : null
}

output "custom_model_service" {
  description = "Kubernetes service name for the custom model"
  value       = var.deploy_custom_model ? "${var.custom_model_name}-service" : null
}

output "model_registry_id" {
  description = "ID of the model registry (if custom model is deployed)"
  value       = var.deploy_custom_model ? castai_ai_optimizer_model_registry.custom_models[0].id : null
}
