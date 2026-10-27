# List all roles in the organization and index them by name. Role names are
# unique per organization, so the map has one entry per name. To bind a single
# role into a resource, prefer the castai_role data source, which looks a role
# up by name directly.
data "castai_roles" "all" {
  organization_id = var.organization_id
}

locals {
  roles_by_name = { for role in data.castai_roles.all.roles : role.name => role }
}

output "viewer_role_id" {
  value = local.roles_by_name["Viewer"].id
}
