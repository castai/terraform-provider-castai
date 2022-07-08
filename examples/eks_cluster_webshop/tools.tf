resource "helm_release" "kube_prometheus_stack" {
  name             = "prom-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  namespace        = "tools"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true
  version          = "35.2.0"

  values = [
    templatefile("./helm-values/kube-prometheus-stack.yaml.tpl", { grafana_password = random_password.grafana_admin.result })
  ]

}

resource "random_password" "grafana_admin" {
  length           = 7
  override_special = ""
}

resource "helm_release" "loki" {
  name             = "loki"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "loki-simple-scalable"
  namespace        = "tools"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true
  version          = "1.8.0"

  values = [
    templatefile("./helm-values/loki.yaml.tpl", { bucket_name = var.loki_bucket_name, s3_path = "s3://${var.cluster_region}", loki_role_arn = module.eks_iam_role_s3.service_account_role_arn})
  ]

  depends_on = [helm_release.kube_prometheus_stack]
}

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  namespace        = "cert-manager"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "installCRDs"
    value = "true"
  }
}

# terraform-provider-kubernetes has a bug https://github.com/hashicorp/terraform-provider-kubernetes/issues/1367#issuecomment-1023327627
resource "kubectl_manifest" "self_signed" {
  yaml_body  = <<YAML
apiVersion: "cert-manager.io/v1"
kind: "ClusterIssuer"
metadata:
  name: "selfsigned"
spec:
   selfSigned: {}
YAML
  depends_on = [helm_release.cert_manager]
}

resource "kubectl_manifest" "clusterissuer_letsencrypt_prod" {
  depends_on = [helm_release.cert_manager]
  yaml_body  = <<YAML
apiVersion: "cert-manager.io/v1"
kind: ClusterIssuer
metadata:
  name: "letsencrypt-prod"
spec:
  acme:
    preferredChain: ""
    privateKeySecretRef:
        name: "letsencrypt-prod"
    server: "https://acme-v02.api.letsencrypt.org/directory"
    solvers:
      - http01:
          ingress:
            class: nginx
YAML
}
