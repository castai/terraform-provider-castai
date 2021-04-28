resource "castai_credentials" "gcp" {
  name = "cicd-gcp"
  gcp {
    service_account_json = base64decode(google_service_account_key.cicd.private_key)
  }
}

data "castai_credentials" "do" {
  name = "cicd-do"
}

resource "castai_cluster" "cicd" {
  name   = "cicd"
  region = "us-east"
  credentials = [
    castai_credentials.gcp.id,
    data.castai_credentials.do.id
  ]

  initialize_params {
    nodes {
      cloud = "gcp"
      role  = "master"
      shape = "medium"
    }
    nodes {
      cloud = "gcp"
      role  = "master"
      shape = "medium"
    }
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
}

data "kubernetes_service" "ingress" {
  metadata {
    name = "ingress-nginx-controller"
    namespace = "ingress-nginx"
  }
}
