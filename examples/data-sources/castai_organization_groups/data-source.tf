# List groups filtered by how they are managed. SCIM-synced groups come from
# the identity provider and only exist on the CAST AI side, so managed_by =
# ["idp-sync"] surfaces exactly those. An empty result is an empty list.
data "castai_organization_groups" "idp" {
  organization_id = var.organization_id
  managed_by      = ["idp-sync"]
}

locals {
  idp_groups_by_name = { for g in data.castai_organization_groups.idp.groups : g.name => g }
}

output "platform_group_id" {
  value = local.idp_groups_by_name["platform-engineers"].id
}
