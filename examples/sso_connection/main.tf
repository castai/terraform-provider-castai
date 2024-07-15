resource "castai_sso_connection" "sso" {
  name                     = "azure_sso"
  email_domain             = azuread_application.castai_sso.publisher_domain
  additional_email_domains = ["example.com", "example.net"]
  aad {
    client_id     = azuread_application.castai_sso.client_id
    client_secret = azuread_application_password.castai_sso.value
    ad_domain     = azuread_application.castai_sso.publisher_domain
  }
  depends_on = [time_sleep.wait_10_seconds]
}

# Since creating castai_sso_connection immediately using azuread credentials fails with
# 'The identity of the calling application could not be established' or
# 'ClientSecretCredential authentication failed' for the sake of this example
# we will use a simple sleep to ensure this example will work out of the box.
resource "time_sleep" "wait_10_seconds" {
  depends_on = [
    azuread_app_role_assignment.this,
    azuread_application_password.castai_sso
  ]

  create_duration = "10s"
}
