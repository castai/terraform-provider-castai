# Model Registry for custom models (S3 bucket).
resource "castai_ai_optimizer_model_registry" "custom_models" {
  count = var.deploy_custom_model ? 1 : 0

  bucket = var.model_registry_bucket
  region = var.model_registry_region
  prefix = "models/"
}

# Custom private model specs.
resource "castai_ai_optimizer_model_specs" "custom_model" {
  count = var.deploy_custom_model ? 1 : 0

  model         = var.custom_model_name
  registry_type = "PRIVATE"

  private_registry {
    base_model_id = var.custom_model_name
    registry_id   = castai_ai_optimizer_model_registry.custom_models[0].id
  }
}

# Hosted model deployment for custom private model.
resource "castai_ai_optimizer_hosted_model" "custom_model" {
  count = var.deploy_custom_model ? 1 : 0

  cluster_id     = castai_eks_clusterid.cluster_id.id
  model_specs_id = castai_ai_optimizer_model_specs.custom_model[0].id
  service        = "${var.custom_model_name}-service"
  port           = 8080

  # No vllm_config needed for private models.

  # Horizontal autoscaling configuration.
  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 2
    target_metric = "GPU_CACHE_USAGE_PERCENTAGE"
    target_value  = 5
  }

  depends_on = [module.castai-eks-cluster]
}
