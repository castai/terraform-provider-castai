data "cloudflare_zones" "cicd" {
  filter {
    name = var.cloudflare_zone
  }
}

resource "cloudflare_record" "argocd" {
  zone_id = lookup(data.cloudflare_zones.cicd.zones[0], "id")
  name    = var.cloudflare_argocd_subdomain
  value   = data.kubernetes_service.ingress.status.0.load_balancer.0.ingress.0.hostname
  type    = "CNAME"
  ttl     = 1
  proxied = false
}

resource "cloudflare_record" "argocd_auth" {
  zone_id = lookup(data.cloudflare_zones.cicd.zones[0], "id")
  name    = "authenticate.${var.cloudflare_argocd_subdomain}"
  value   = data.kubernetes_service.ingress.status.0.load_balancer.0.ingress.0.hostname
  type    = "CNAME"
  ttl     = 1
  proxied = false
}

resource "kubernetes_namespace" "argo" {
  metadata {
    name = "argo"
  }
}

resource "kubernetes_secret" "charts_repo_creds" {
  metadata {
    name      = "charts-repo-creds"
    namespace = kubernetes_namespace.argo.metadata[0].name
  }
  data = {
    user = var.charts_user,
    pass = var.charts_pass
  }
}

resource "kubernetes_secret" "gitlab_git_access" {
  metadata {
    name      = "gitlab-git-access"
    namespace = kubernetes_namespace.argo.metadata[0].name
  }
  data = {
    sshPrivateKey = base64decode(var.helmChartsSSHPrivateKey),
  }
}

resource "kubernetes_secret" "cluster_dev_master" {
  metadata {
    name      = "cluster-dev-master"
    namespace = kubernetes_namespace.argo.metadata[0].name
    labels = {
      "argocd.argoproj.io/secret-type" = "cluster"
    }
  }
  data = {
    name = var.argocd_dev_cluster_name,
    server = var.argocd_dev_cluster_server,
    config = base64decode(var.argocd_dev_cluster_config)
  }
}

resource "kubernetes_secret" "cluster_prod_master" {
  metadata {
    name      = "cluster-prod-master"
    namespace = kubernetes_namespace.argo.metadata[0].name
    labels = {
      "argocd.argoproj.io/secret-type" = "cluster"
    }
  }
  data = {
    name = var.argocd_prod_cluster_name,
    server = var.argocd_prod_cluster_server,
    config = base64decode(var.argocd_prod_cluster_config)
  }
}

resource "helm_release" "argocd" {
  name       = "argocd"
  namespace  = kubernetes_namespace.argo.metadata[0].name
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "3.1.0"
  values = [file("argocd-values.yaml")]
  depends_on = [
    kubernetes_secret.charts_repo_creds,
    kubernetes_secret.gitlab_git_access,
    kubernetes_secret.cluster_dev_master,
    cloudflare_record.argocd,
    cloudflare_record.argocd_auth
  ]

  set {
    name = "server.additionalProjects[0].destinations[0].server"
    value = var.argocd_dev_cluster_server
  }

  set {
    name = "server.additionalProjects[1].destinations[0].server"
    value = var.argocd_prod_cluster_server
  }
}
