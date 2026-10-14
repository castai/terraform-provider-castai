data "castai_organization" "test" {
  name = "My test organization name"
}

data "castai_roles" "owner" {
  organization_id = data.castai_organization.test.id
  name            = "Owner"
  type            = "DEFAULT"
}

data "castai_roles" "viewer" {
  organization_id = data.castai_organization.test.id
  name            = "Viewer"
  type            = "DEFAULT"
}

resource "castai_role_bindings" "owner_test" {
  organization_id = data.castai_organization.test.id
  name            = "Role binding owner"
  description     = "Owner access for whole organization."

  role_id = one(data.castai_roles.owner.roles[*].id)

  lifecycle {
    precondition {
      condition     = length(data.castai_roles.owner.roles) == 1
      error_message = "Expected 1 role 'Owner', got: ${jsonencode(data.castai_roles.owner.roles)}"
    }
  }

  scopes {
    kind        = "organization"
    resource_id = data.castai_organization.test.id
  }
  subjects {
    subject {
      kind    = "user"
      user_id = "21c133e2-a899-4f51-b297-830bc62e51d6" # user x
    }
    subject {
      kind    = "user"
      user_id = "0d1efe35-7ecb-4821-a52d-fd56c9710a64" # user y
    }
    subject {
      kind     = "group"
      group_id = "651734a7-0d0c-49f3-9654-dd92175febaa"
    }
    subject {
      kind               = "service_account"
      service_account_id = "3bf49513-3e9c-4a12-962c-af3bb1a85074"
    }
  }
}

resource "castai_role_bindings" "viewer_test" {
  organization_id = data.castai_organization.test.id
  name            = "Role binding viewer for cluster 7063d31c-897e-48ef-a322-bdfda6fdbcfb"
  description     = "Viewer access for on of the clusters."

  role_id = one(data.castai_roles.viewer.roles[*].id)

  lifecycle {
    precondition {
      condition     = length(data.castai_roles.viewer.roles) == 1
      error_message = "Expected 1 role 'Viewer', got: ${jsonencode(data.castai_roles.viewer.roles)}"
    }
  }

  scopes {
    kind        = "cluster"
    resource_id = "7063d31c-897e-48ef-a322-bdfda6fdbcfb"
  }
  scopes {
    kind        = "cluster"
    resource_id = "9872e54f-1122-33ef-b456-cdef98765432"
  }
  subjects {
    subject {
      kind    = "user"
      user_id = "21c133e2-a899-4f51-b297-830bc62e51d6" # user z
    }
  }
}
