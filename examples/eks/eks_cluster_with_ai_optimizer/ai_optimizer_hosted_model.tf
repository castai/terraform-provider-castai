# Hosted model deployment for a predefined CastAI-managed model.
# The model_specs_id references a pre-existing model specs entry managed by CastAI.
# Obtain the ID from the CAST AI console or API — no castai_ai_optimizer_model_specs resource is needed.
resource "castai_ai_optimizer_hosted_model" "llama_3_1_8b" {
  count = var.deploy_predefined_model ? 1 : 0

  cluster_id         = castai_eks_clusterid.cluster_id.id
  model_specs_id     = "099581b9-fe24-4021-b11b-52c513eb44b8" # Predefined model specs ID for Llama 3.1 8B Instruct
  service            = "llama3-1-8b"
  port               = 11434
  node_template_name = "llms-by-castai"

  vllm_config {
    hugging_face_token = var.hf_token
  }

  # Horizontal autoscaling configuration.
  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 3
    target_metric = "GPU_CACHE_USAGE_PERCENTAGE"
    target_value  = 50
  }

  # Hibernation configuration — automatically scale down the model when idle
  # and resume it when traffic returns.
  hibernation {
    enabled = true

    hibernate_condition {
      duration      = "1800s"
      request_count = 1
    }

    resume_condition {
      duration      = "600s"
      request_count = 1
    }
  }

  depends_on = [module.castai-eks-cluster]
}
