data "castai_organization" "dev" {
  provider = castai.dev
  name     = var.castai_dev_organization_name
}

data "castai_organization" "prod" {
  provider = castai.dev
  name     = var.castai_prod_organization_name
}

resource "castai_organization_members" "dev" {
  provider        = castai.dev
  organization_id = data.castai_organization.dev.id

  owners = [
    "owner@test.ai",
  ]

  members = [
    "member@test.ai",
  ]

  viewers = []
}

resource "castai_organization_members" "prod" {
  provider        = castai.prod
  organization_id = data.castai_organization.prod.id

  owners = [
    "owner@test.ai",
  ]

  members = []

  viewers = [
    "viewer@test.ai",
  ]
}


