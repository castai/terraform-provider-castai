provider "castai" {
  api_token = var.castai_api_token
}

resource "castai_credentials" "example_gcp" {
  name = "example-gcp"
  gcp {
    service_account_json = var.gcp_service_account_json
  }
}

resource "castai_credentials" "example_aws" {
  name = "example-aws"
  aws {
    access_key_id     = var.aws_access_key_id
    secret_access_key = var.aws_secret_access_key
  }
}

resource "castai_credentials" "example_azure" {
  name = "example-azure"
  azure {
    service_principal_json = var.azure_service_principal_json
  }
}

resource "castai_cluster" "example_cluster" {
  name   = "example-cluster"
  region = "eu-central"
  credentials = [
    castai_credentials.example_gcp.id,
    castai_credentials.example_aws.id,
  ]

  initialize_params {
    nodes {
      cloud = "aws"
      role  = "master"
      shape = "medium"
    }
    nodes {
      cloud = "aws"
      role  = "worker"
      shape = "small"
    }
    nodes {
      cloud = "gcp"
      role  = "worker"
      shape = "small"
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
      clouds = ["gcp","aws"]
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
