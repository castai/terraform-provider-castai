resource "castai_service_account" "service_account" {
  organization_id = organization.id
  name            = "example-service-account"
  description     = "service account description"
}
