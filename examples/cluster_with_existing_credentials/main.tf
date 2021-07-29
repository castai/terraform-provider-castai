provider "castai" {
  api_token = "<<replace-this-with-your-api-token>>"
}

data "castai_credentials" "existing_gcp" {
  name = "existing-gcp"
}

resource "castai_cluster" "example_cluster" {
  name = "example-cluster"
  region = "eu-central"
  credentials = [
    data.castai_credentials.existing_gcp.id
  ]

  initialize_params {
    nodes {
      cloud = "gcp"
      role = "master"
      shape = "medium"
    }
    nodes {
      cloud = "gcp"
      role = "worker"
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
        delay_seconds = 120
      }
    }

    spot_instances {
      clouds = [
        "gcp",
        "aws"]
      enabled = false
    }

    unschedulable_pods {
      enabled = false
      headroom {
        enabled = false
        cpu_percentage = 10
        memory_percentage = 10
      }
      node_constraints {
        enabled = true
        max_node_cpu_cores = 32
        max_node_ram_gib = 256
        min_node_cpu_cores = 2
        min_node_ram_gib = 8
      }
    }
  }
}

output "example_cluster_kubeconfig" {
  value = castai_cluster.example_cluster.kubeconfig.0.raw_config
}