resource "castai_commitments" "gcp_test" {
  gcp_cuds_json = file("./cuds.json")
  commitment_configs {
    matcher {
      region = "us-east4"
      type   = "COMPUTE_OPTIMIZED_C2D"
      name   = "test"
    }
    prioritization = true
    allowed_usage  = 0.6
    status         = "Inactive"

    assignments {
      cluster_id = "cluster-id-1" # priority 1 cluster - prioritization is enabled
    }
    assignments {
      cluster_id = "cluster-id-2" # priority 2 cluster - prioritization is enabled
    }
  }
}

resource "castai_commitments" "azure_test" {
  azure_reservations_csv = file("./reservations.csv")
  commitment_configs {
    matcher {
      region = "eastus"
      type   = "Standard_D32as_v4"
      name   = "test-res-1"
    }
    prioritization = false
    allowed_usage  = 0.9
    status         = "Active"

    assignments {
      cluster_id = "cluster-id-3"
    }
    assignments {
      cluster_id = "cluster-id-4"
    }
  }
}
