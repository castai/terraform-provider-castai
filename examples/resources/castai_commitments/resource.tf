resource "castai_commitments" "gcp_test" {
  gcp_cuds_json = file("./cuds.json")
  gcp_cud_configs = [
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
