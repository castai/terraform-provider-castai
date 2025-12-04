module "load_balancer_controller" {
  source  = "DNXLabs/eks-lb-controller/aws"
  version = "0.6.0"

  cluster_identity_oidc_issuer     = module.eks.cluster_oidc_issuer_url
  cluster_identity_oidc_issuer_arn = module.eks.oidc_provider_arn
  cluster_name                     = module.eks.cluster_name

  helm_chart_version = "1.4.1"
}

resource "helm_release" "nginx" {
  name             = "nginx-ingress"
  repository       = "https://kubernetes.github.io/ingress-nginx"
  chart            = "ingress-nginx"
  namespace        = "ingress-nginx"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true
  version          = "4.1.1"

  values = [
    templatefile("./helm-values/ingress-nginx.yaml", { ingress_id = join(",", [for ip in aws_eip.this : ip.id]) })
  ]

  depends_on = [helm_release.kube_prometheus_stack]
}

resource "aws_eip" "this" {
  count = 3
}

