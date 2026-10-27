# Resolve the Viewer role ID by name instead of hardcoding the role UUID.
# Role names are unique per organization, so the name identifies the role.
data "castai_role" "viewer" {
  organization_id = var.organization_id
  name            = "Viewer"
}

resource "castai_role_bindings" "viewer_test" {
  organization_id = var.organization_id
  name            = "Role binding viewer"

  role_id = data.castai_role.viewer.id

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
