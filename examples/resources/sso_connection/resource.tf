resource "castai_sso_connection" "aad" {
  name         = "aad_connection"
  email_domain = "aad_connection@test.com"
  aad {
    client_id     = azuread_application.castai_sso.client_id
    client_secret = azuread_application_password.castai_sso.value
    ad_domain     = azuread_application.castai_sso.publisher_domain
  }
}

resource "castai_sso_connection" "keycloak" {
  name         = "keycloak_connection"
  email_domain = "example.com"
  oidc {
    issuer_url    = "https://keycloak.example.com/realms/my-realm"
    client_id     = "castai"
    client_secret = var.keycloak_client_secret
  }
}
