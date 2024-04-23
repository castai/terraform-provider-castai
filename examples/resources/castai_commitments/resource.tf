resource "castai_commitments" "gcp_test" {
  gcp_cuds_json = file("./cuds.json")
  gcp_cud_configs = [
    {
      matcher = {
        region = "us-east4"
        type = "COMPUTE_OPTIMIZED_C2D"
        name = "test"
      }
      prioritization = true
      allowed_usage = 0.6
      status = "Inactive"
    }
  ]
}
