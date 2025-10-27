resource "castai_enterprise_role_binding" "enterprise_rb" {
  enterprise_id   = "63a24904-9ae7-4a6b-bdb2-15dcafc42b66"
  organization_id = "63a24904-9ae7-4a6b-bdb2-15dcafc42b66"

  name        = "Enterprise Role Binding - Owner"
  description = "Enterprise Role Binding for Owner role"
  role_id     = "0145fead-6c09-41cb-a262-cfaaab392971"

  scopes {
    organization {
      id = "63a24904-9ae7-4a6b-bdb2-15dcafc42b66"
    }
  }

  subjects {
    user {
      id = "67e71067-04c4-4423-8655-9c366b7234b0"
    }
  }
}

resource "castai_enterprise_role_binding" "child_rb" {
  enterprise_id   = "63a24904-9ae7-4a6b-bdb2-15dcafc42b66"
  organization_id = "76452f30-9b42-4847-a33e-9bc39e80ead2"

  name        = "Child Role Binding - Member"
  description = "Child Role Binding for Member role"
  role_id     = "8c60bd8e-21de-402a-969f-add07fd22c1b"

  scopes {
    organization {
      id = "76452f30-9b42-4847-a33e-9bc39e80ead2"
    }
  }

  subjects {
    user {
      id = "67e71067-04c4-4423-8655-9c366b7234b0"
    }

    service_account {
      id = "7ae72c7f-c0ac-4e15-8287-cc058b43eb14"
    }

    service_account {
      id = "8dd98f12-43f3-48f5-89b3-d0ca96ec00b2"
    }

    service_account {
      id = "66be7d78-7081-41d8-9549-97fd5767801b"
    }

    group {
      id = "0fbeba03-d1c0-48b9-8deb-09da9f43a75b"
    }
  }
}