resource "castai_security_runtime_rule" "example" {
  name              = "Example rule: DNS to crypto mining"
  severity          = "SEVERITY_LOW"
  enabled           = false
  rule_engine_type  = "RULE_ENGINE_TYPE_CEL"
  rule_text         = <<EOT
event.type == event_dns && event.dns.network_details.category == category_crypto
EOT
  resource_selector = <<EOT
resource.namespace == "default"
EOT
  labels = {
    environment = "dev"
    team        = "security"
  }
}
