# Resolve built-in role IDs by name instead of hardcoding role UUIDs.
data "castai_roles" "viewer" {
  organization_id = var.organization_id
  name            = "Viewer"
  type            = "DEFAULT"
}

# one() fails the plan when the name matches multiple roles (role names are
# not guaranteed to be unique), so bind the role safely:
resource "castai_role_bindings" "viewer_test" {
  organization_id = var.organization_id
  name            = "Role binding viewer"

  role_id = one(data.castai_roles.viewer.roles[*].id)

  lifecycle {
    precondition {
      condition     = length(data.castai_roles.viewer.roles) == 1
      error_message = "Expected 1 role 'Viewer', got: ${jsonencode(data.castai_roles.viewer.roles)}"
    }
  }

  scopes {
    kind        = "cluster"
    resource_id = var.cluster_id
  }

  subjects {
    subject {
      kind    = "user"
      user_id = var.user_id
    }
  }
}
