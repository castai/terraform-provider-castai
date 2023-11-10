resource "castai_sso_connection" "sso" {
  name         = "aad_connection"
  email_domain = "aad_connection@test.com"
  aad {
    client_id     = azuread_application.castai_sso.client_id
    client_secret = azuread_application_password.castai_sso.value
    ad_domain     = azuread_application.castai_sso.publisher_domain
  }
}
