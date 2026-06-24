terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
  }
  required_version = ">= 0.13"
}


resource "castai_enterprise_service_account" "ci" {
  enterprise_id   = "47a7f128-afca-4fb5-ba93-f6a98e370ccf"
  organization_id = "50e2f997-c717-4669-9c23-504d10f5e515"
  name            = "enterprise-service-account"
  description     = "Service account for CI pipelines"
}

