# Terragrunt users typically generate this file through a `generate "provider"` block in
# their root terragrunt.hcl. It is kept here so the module can also be used with plain
# Terraform/OpenTofu. See examples/terragrunt/terragrunt.hcl for the generated variant.
provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "aws" {
  region  = var.aws_cluster_region
  profile = var.profile
}
