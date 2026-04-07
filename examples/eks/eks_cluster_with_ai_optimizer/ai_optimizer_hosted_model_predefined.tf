# Predefined HuggingFace model specs (Llama 3.1 8B Instruct).
resource "castai_ai_optimizer_model_specs" "llama_3_1_8b" {
  count = var.deploy_predefined_model ? 1 : 0

  model         = "llama-3.1-8b-instruct"
  registry_type = "HUGGING_FACE"
  type          = "chat"
  routable      = true

  huggingface {
    model_name = "meta-llama/Llama-3.1-8B-Instruct"
  }
}

# Hosted model deployment for predefined HuggingFace model.
resource "castai_ai_optimizer_hosted_model" "llama_3_1_8b" {
  count = var.deploy_predefined_model ? 1 : 0

  cluster_id     = castai_eks_clusterid.cluster_id.id
  model_specs_id = castai_ai_optimizer_model_specs.llama_3_1_8b[0].id
  service        = "llama31-service"
  port           = 8080

  vllm_config {
    secret_name = var.hf_token_secret_name
  }

  # Horizontal autoscaling configuration.
  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 4
    target_metric = "REQUESTS_PER_SECOND"
    target_value  = 10
  }

  depends_on = [module.castai-eks-cluster]
}
