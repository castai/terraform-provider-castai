resource "castai_ai_optimizer_model_registry" "example" {
  provider_type = "S3"
  credentials   = var.registry_credentials

  s3 {
    bucket = "my-company-model-registry"
    region = "us-east-1"
    prefix = "models/"
  }
}
