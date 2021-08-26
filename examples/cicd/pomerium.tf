resource "kubernetes_namespace" "pomerium" {
  metadata {
    name = "pomerium"
  }
}

resource "helm_release" "pomerium" {
  name       = "pomerium"
  namespace  = kubernetes_namespace.pomerium.metadata[0].name
  repository = "https://helm.pomerium.io"
  chart      = "pomerium"
  version    = "16.0.2"
  values     = [file("pomerium-values.yaml")]

  set {
    name  = "authenticate.idp.clientID"
    value = var.gcp_auth_client_id
  }

  set {
    name  = "authenticate.idp.clientSecret"
    value = var.gcp_auth_client_secret
  }
}
