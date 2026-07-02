resource "castai_enterprise_service_account" "example" {
  enterprise_id   = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
  organization_id = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
  name            = "example-service-account"
  description     = "Service account managed by Terraform"
}
