# HuggingFace hosted model with optional autoscaling
resource "castai_ai_optimizer_hosted_model" "hf_example" {
  cluster_id     = castai_eks_cluster.example.id
  model_specs_id = castai_ai_optimizer_model_specs.hf_example.id
  service        = "llama31"
  port           = 8080

  vllm_config {
    secret_name = "huggingface-token"
  }

  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 4
    target_metric = "REQUESTS_PER_SECOND"
    target_value  = 10
  }
}

# Private hosted model (no vllm_config needed)
resource "castai_ai_optimizer_hosted_model" "private_example" {
  cluster_id     = castai_eks_cluster.example.id
  model_specs_id = castai_ai_optimizer_model_specs.private_example.id
  service        = "custom-model"
  port           = 8080
}
