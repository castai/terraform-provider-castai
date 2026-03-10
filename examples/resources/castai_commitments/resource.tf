# GCP example with default OVERWRITE mode and auto_assignment enabled (default)
resource "castai_commitments" "gcp_test" {
  gcp_cuds_json = file("./cuds.json")
  commitment_configs {
    matcher {
      region = "us-east4"
      type   = "COMPUTE_OPTIMIZED_C2D"
      name   = "test"
    }
    prioritization   = true
    allowed_usage    = 0.6
    status           = "Inactive"
    scaling_strategy = "Default"

    assignments {
      cluster_id = "cluster-id-1" # priority 1 cluster - prioritization is enabled
    }
    assignments {
      cluster_id = "cluster-id-2" # priority 2 cluster - prioritization is enabled
    }
  }
}

# Azure example with default OVERWRITE mode and auto_assignment enabled (default)
resource "castai_commitments" "azure_test" {
  azure_reservations_csv = file("./reservations.csv")
  commitment_configs {
    matcher {
      region = "eastus"
      type   = "Standard_D32as_v4"
      name   = "test-res-1"
    }
    prioritization   = false
    allowed_usage    = 0.9
    status           = "Active"
    scaling_strategy = "Default"

    assignments {
      cluster_id = "cluster-id-3"
    }
    assignments {
      cluster_id = "cluster-id-4"
    }
  }
}

# Azure example with APPEND mode and auto_assignment disabled to prevent commitments from being auto-assigned to all matching clusters.
resource "castai_commitments" "team_a" {
  azure_reservations_csv = file("./team-a-reservations.csv")
  import_mode            = "APPEND"

  commitment_configs {
    matcher {
      region = "eastus"
      type   = "Standard_DS2_v2"
      name   = "team-a-reservation"
    }
    prioritization   = false
    allowed_usage    = 1
    status           = "Active"
    scaling_strategy = "Default"
    auto_assignment  = false

    assignments {
      cluster_id = "team-a-cluster-id"
    }
  }
}
