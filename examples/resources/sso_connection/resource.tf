resource "castai_sso_connection" "aad" {
  name            = "aad_connection"
  organization_id = var.organization_id
  email_domain    = "aad_connection@test.com"
  aad {
    client_id     = azuread_application.castai_sso.client_id
    client_secret = azuread_application_password.castai_sso.value
    ad_domain     = azuread_application.castai_sso.publisher_domain
  }
}

resource "castai_sso_connection" "keycloak" {
  name              = "keycloak_connection"
  organization_id   = var.organization_id
  email_domain      = "example.com"
  block_invitations = true
  oidc {
    issuer_url    = "https://keycloak.example.com/realms/my-realm"
    client_id     = "castai"
    client_secret = var.keycloak_client_secret
  }
}

# DNS TXT records to publish for email-domain verification (one per claimed
# domain that is still pending). CAST AI re-checks DNS automatically until each
# claim is verified.
output "email_domain_verification_records" {
  value = {
    for v in castai_sso_connection.keycloak.email_domain_validation :
    v.domain => { name = v.txt_record_name, value = v.txt_record_value, status = v.status }
    if v.txt_record_name != ""
  }
}
