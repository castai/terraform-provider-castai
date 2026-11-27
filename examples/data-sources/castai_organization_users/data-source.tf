# Look up users by email. The API has no email filter and one email can map to
# several identities (e.g. a password login and an SSO login), so the data
# source is filtered client-side and can return multiple users. scim_active is
# only applied when explicitly set.
data "castai_organization_users" "alice" {
  organization_id = var.organization_id
  email           = "alice@corp.com"
  scim_active     = true
}

# Use the returned user id as a subject for role bindings. The lifecycle
# precondition fails the plan when the email matches more than one identity.
resource "castai_role_bindings" "alice" {
  organization_id = var.organization_id
  role_id         = castai_role.viewer.id

  subject {
    kind    = "user"
    user_id = one(data.castai_organization_users.alice.users[*].id)
  }

  lifecycle {
    precondition {
      condition     = length(data.castai_organization_users.alice.users) == 1
      error_message = "Expected 1 identity for alice@corp.com, got: ${jsonencode(data.castai_organization_users.alice.users)}"
    }
  }
}
