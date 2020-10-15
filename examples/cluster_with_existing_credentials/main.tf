provider "castai" {
  version = "0.0.1"
  api_url = "https://console.dev-master.cast.ai/api/"
  api_token = "<<replace-this-with-your-api-token"
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
      shape = "small"
    }
    nodes {
      cloud = "gcp"
      role = "worker"
      shape = "small"
    }
  }
}

output "example_cluster_kubeconfig" {
  value = castai_cluster.example_cluster.kubeconfig
}
