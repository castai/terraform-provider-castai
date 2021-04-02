provider "castai" {
  api_token = "<<replace-this-with-your-api-token>>"
}

data "castai_credentials" "existing_gcp" {
  name = "existing-gcp"
}

resource "castai_cluster" "example_cluster" {
  name   = "example-cluster"
  region = "eu-central"
  credentials = [
    data.castai_credentials.existing_gcp.id
  ]

  initialize_params {
    nodes {
      cloud = "gcp"
      role  = "master"
      shape = "medium"
    }
    nodes {
      cloud = "gcp"
      role  = "worker"
      shape = "medium"
    }
  }

  autoscaler_policies {

    cluster_limits {
      cpu {
        max_cores = 20
        min_cores = 2
      }
    }

    node_downscaler {
      empty_nodes {
        enabled = false
      }
    }

    spot_instances {
      clouds = ["gcp"]
      enabled = false
    }

    unschedulable_pods {
      enabled = false
      headroom {
        cpu_percentage = 10
        memory_percentage = 10
      }
    }
  }
}

output "example_cluster_kubeconfig" {
  value = castai_cluster.example_cluster.kubeconfig.0.raw_config
}