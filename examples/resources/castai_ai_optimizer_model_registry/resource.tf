resource "castai_ai_optimizer_model_registry" "example" {
  bucket = "my-company-model-registry"
  region = "us-east-1"
  prefix = "models/"
}
