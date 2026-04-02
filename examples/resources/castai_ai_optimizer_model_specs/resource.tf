# HuggingFace model specs
resource "castai_ai_optimizer_model_specs" "hf_example" {
  model         = "llama-3.1-8b-instruct"
  registry_type = "HUGGING_FACE"
  type          = "chat"
  routable      = true

  huggingface {
    model_name = "meta-llama/Llama-3.1-8B-Instruct"
  }
}

# Private (S3) model specs
resource "castai_ai_optimizer_model_specs" "private_example" {
  model         = "my-custom-model"
  registry_type = "PRIVATE"

  private_registry {
    base_model_id = "my-custom-model"
    registry_id   = castai_ai_optimizer_model_registry.example.id
  }
}
