resource "castai_service_account" "service_account" {
  organization_id = organization.id
  name = "service-account-name"
  description = "service account description"
}
