resource "castai_commitments" "gcp_test" {
  gcp_cuds_json = file("./cuds.json")
  commitment_configs = [
    {
      match_region   = "us-east4"
      match_type     = "COMPUTE_OPTIMIZED_C2D"
      match_name     = "test"
      prioritization = true
      allowed_usage  = 0.6
      status         = "Inactive"
    }
  ]
}

resource "castai_commitments" "azure_test" {
  azure_reservations_csv = file("./reservations.csv")
  commitment_configs = [
    {
      match_region   = "eastus"
      match_type     = "Standard_D32as_v4"
      match_name     = "test-res-1"
      prioritization = false
      allowed_usage  = 0.9
      status         = "Active"
    }
  ]
}
