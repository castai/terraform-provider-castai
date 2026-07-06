# Example 1: Create an enterprise service account scoped to a child organization.
resource "castai_enterprise_service_account" "child_org" {
  enterprise_id   = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
  organization_id = "50e2f997-c717-4669-9c23-504d10f5e515"
  name            = "example-child-org-service-account"
  description     = "Service account managed by Terraform in a child organization"
}

# Example 2: Create an enterprise service account scoped to the enterprise organization itself.
# When organization_id is omitted it defaults to enterprise_id (enterprise scope).
resource "castai_enterprise_service_account" "enterprise_org" {
  enterprise_id = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
  name          = "example-enterprise-service-account"
  description   = "Service account managed by Terraform in the enterprise organization"
}
