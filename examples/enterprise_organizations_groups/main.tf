resource "castai_enterprise_group" "group1" {
  enterprise_id = "3e81b0c1-ea70-4513-8c11-60260fa04fba"

  name            = "Enterprise Group 1"
  description     = "A description of the first group."
  organization_id = "3e81b0c1-ea70-4513-8c11-60260fa04fba"

  members {
    member {
      kind = "user"
      id   = "c371d6a4-b6d3-4052-8742-1119013392b9"
    }
  }

  role_bindings {
    role_binding {
      name    = "Enterprise Group 1 - Viewer"
      role_id = "d2ff8433-af9a-4a39-8900-6f6b96e43eb6"
      scopes {
        scope {
          organization = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
        }
      }
    }

    role_binding {
      name    = "Enterprise Group 1 - Owner"
      role_id = "0145fead-6c09-41cb-a262-cfaaab392971"
      scopes {
        scope {
          organization = "3e81b0c1-ea70-4513-8c11-60260fa04fba"
        }
      }
    }
  }
}

# Looks up the built-in Member role of the child organization by its unique name.
data "castai_role" "child_member" {
  organization_id = "d83b788d-bc9d-4bfa-947e-8d299f3d5852"
  name            = "Member"
}

resource "castai_enterprise_group" "group2" {
  enterprise_id = "3e81b0c1-ea70-4513-8c11-60260fa04fba"

  name            = "Enterprise Group 2"
  description     = "A description of the second group"
  organization_id = "d83b788d-bc9d-4bfa-947e-8d299f3d5852"

  members {
    member {
      kind = "user"
      id   = "c371d6a4-b6d3-4052-8742-1119013392b9"
    }

    member {
      kind = "service_account"
      id   = "e498f109-eb63-43d1-94e5-6e6d2232ffab"
    }
  }

  role_bindings {
    role_binding {
      name    = "Enterprise Group 2 - Member"
      role_id = data.castai_role.child_member.id
      scopes {
        scope {
          organization = "d83b788d-bc9d-4bfa-947e-8d299f3d5852"
        }
      }
    }
  }
}
