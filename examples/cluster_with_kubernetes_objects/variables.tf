variable "castai_api_token" {
  default = "<<replace-this-with-your-token>>"
}

variable "gcp_service_account_json" {
  default = <<EOF
{
  "type": "service_account",
  "project_id": "<<replace-this-json-with-your-credentials>>",
  "private_key_id": "<<replace-this-json-with-your-credentials>>",
  "private_key": "<<replace-this-json-with-your-credentials>>",
  "client_email": "<<replace-this-json-with-your-credentials>>",
  "client_id": "<<replace-this-json-with-your-credentials>>",
  "auth_uri": "<<replace-this-json-with-your-credentials>>",
  "token_uri": "<<replace-this-json-with-your-credentials>>",
  "auth_provider_x509_cert_url": "<<replace-this-json-with-your-credentials>>",
  "client_x509_cert_url": "<<replace-this-json-with-your-credentials>>"
}
EOF
}

variable "azure_service_principal_json" {
  default = <<EOF
{
   "tenantId":"<<replace-this-json-with-your-credentials>>",
   "clientId":"<<replace-this-json-with-your-credentials>>",
   "clientSecret":"<<replace-this-json-with-your-credentials>>",
   "subscriptionId":"<<replace-this-json-with-your-credentials>>"
}
EOF
}

variable "aws_access_key_id" {
  default = "<<replace-this-value-with-your-access-key-id>>"
}

variable "aws_secret_access_key" {
  default = "<<replace-this-value-with-your-access-key>>"
}
