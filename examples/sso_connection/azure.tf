data "azuread_client_config" "current" {}

resource "azuread_application" "castai_sso" {
  display_name = "castai_sso"

  web {
    redirect_uris = ["https://login.cast.ai/login/callback"]
  }

  required_resource_access {
    resource_app_id = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph

    resource_access {
      id   = azuread_service_principal.msgraph.app_role_ids["Directory.Read.All"]
      type = "Role"
    }

    resource_access {
      id   = azuread_service_principal.msgraph.oauth2_permission_scope_ids["User.Read"]
      type = "Scope"
    }
  }
}

resource "azuread_application_password" "castai_sso" {
  application_id = azuread_application.castai_sso.id
}

data "azuread_application_published_app_ids" "well_known" {}

resource "azuread_service_principal" "msgraph" {
  client_id    = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
  use_existing = true
}

resource "azuread_service_principal" "castai_sso" {
  client_id = azuread_application.castai_sso.client_id
}

resource "azuread_app_role_assignment" "this" {
  app_role_id         = azuread_service_principal.msgraph.app_role_ids["Directory.Read.All"]
  principal_object_id = azuread_service_principal.castai_sso.object_id
  resource_object_id  = azuread_service_principal.msgraph.object_id
}
