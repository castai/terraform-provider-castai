# Look up a group by its exact, case-sensitive name. Groups synced from an
# identity provider (SCIM) never enter Terraform state, so the data source
# resolves their UUIDs without hardcoding. It errors when the name matches
# nothing or more than one group.
data "castai_organization_group" "platform" {
  organization_id = var.organization_id
  name            = "platform-engineers"
}

# Bind the group to a role; the resolved id is the group subject.
resource "castai_role_bindings" "platform_viewer" {
  organization_id = var.organization_id
  name            = "platform-viewer"
  role_id         = var.viewer_role_id

  scopes {
    kind        = "organization"
    resource_id = var.organization_id
  }

  subjects {
    subject {
      kind     = "group"
      group_id = data.castai_organization_group.platform.id
    }
  }
}
