data "castai_organization" "dev" {
  name = var.castai_dev_organization_name
}

resource "castai_organization_members" "dev" {
  organization_id = data.castai_organization.dev.id

  owners = [
    "owner@test.ai",
  ]

  members = [
    "member@test.ai",
  ]

  viewers = []
}


