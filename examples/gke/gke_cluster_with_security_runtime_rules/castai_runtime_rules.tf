# 4. Import Runtime security rules.
# Below is example security runtime rule, you can import existing rules from cast ai using fetch_castai_runtime_rules.sh script

resource "castai_security_runtime_rule" "example_rule__dns_to_crypto_mining_" {
  name              = "Example rule: DNS to crypto mining"
  severity          = "SEVERITY_LOW"
  enabled           = false
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
  depends_on = [module.castai-gke-cluster]
}

