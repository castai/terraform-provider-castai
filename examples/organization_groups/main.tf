terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
  }
  required_version = ">= 0.13"
}

data "castai_organization" "test" {
  name = "My test organization name"
}

resource "castai_organization_group" "first_group" {
  organization_id = data.castai_organization.test.id
  name            = "first-group"
  description     = "A description of the first group."

  members {
    member {
      kind  = "user"
      id    = "21c133e2-a899-4f51-b297-830bc62e51d6"
      email = "first-user@cast.ai"
    }
    member {
      kind  = "user"
      id    = "21c133e2-a899-4f51-b297-830bc62e51d7"
      email = "second-user@cast.ai"
    }
    member {
      kind  = "service_account"
      id    = "21c133e2-a899-4f51-b297-830bc62e51d9"
      email = "service_account-2@cast.ai"
    }
  }
}

resource "castai_organization_group" "second_group" {
  organization_id = data.castai_organization.test.id
  name            = "second-group"
  description     = "A description of the second group."

  members {
    member {
      kind  = "user"
      id    = "21c133e2-a899-4f51-b297-830bc62e51d6"
      email = "first-user@cast.ai"
    }
  }
}