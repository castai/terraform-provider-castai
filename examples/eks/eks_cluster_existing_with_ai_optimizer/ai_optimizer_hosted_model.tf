resource "castai_ai_optimizer_hosted_model" "llama_3_2_1b" {
  count = var.deploy_predefined_model ? 1 : 0

  cluster_id         = castai_eks_clusterid.cluster_id.id
  model_specs_id     = "c7a7254f-b7c0-43c5-9a09-5c7afe72de92" # Predefined model specs ID for Llama 3.2 1B Instruct
  service            = "llama3-2-1b"
  port               = 11434
  node_template_name = "llms-by-castai"

  vllm_config {
    hugging_face_token = var.hf_token
  }

  horizontal_autoscaling {
    enabled       = true
    min_replicas  = 1
    max_replicas  = 3
    target_metric = "GPU_CACHE_USAGE_PERCENTAGE"
    target_value  = 50
  }

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

  depends_on = [module.castai_eks_cluster]
}
