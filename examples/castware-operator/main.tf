# Install Castware Operator
resource "helm_release" "castware_operator" {
  name       = "castware-operator"
  namespace  = "castai-agent"
  repository = "https://castai.github.io/helm-charts"
  chart      = "castware-operator"
  version    = var.castware_operator_version

  wait             = true
  wait_for_jobs    = true
  atomic           = true
  cleanup_on_fail  = true
  create_namespace = false # true if you plan to deploy in a cluster without any cast components

  set_sensitive {
    name  = "apiKeySecret.apiKey"
    value = var.castai_api_token
  }

  set {
    name  = "extendedPermissions"
    value = var.extended_permissions
  }

  set {
    name  = "defaultCluster.provider"
    value = var.cluster_provider
  }

  set {
    name  = "defaultCluster.api.apiUrl"
    value = var.castai_api_url
  }

  set {
    name  = "defaultComponents.enabled"
    value = false
  }

  set {
    name  = "defaultCluster.migrationMode"
    value = "autoUpgrade" # or write depends on needs, see README
  }

  # this can be omitted as it's true by default
  set {
    name  = "defaultCluster.terraform"
    value = true
  }

  # add this only if you want to add/you already have a module configured
  depends_on = [
    module.castai-aks-cluster
  ]
}

resource "helm_release" "castware_components" {
  name       = "castware-components"
  namespace  = "castai-agent"
  repository = "https://castai.github.io/helm-charts"
  chart      = "castware-components"

  wait             = true
  wait_for_jobs    = true
  atomic           = true
  cleanup_on_fail  = true
  create_namespace = false

  #overrides
  values = [
    templatefile("${path.module}/castware-values.yaml", {
      aks_cluster_name   = var.aks_cluster_name
      aks_cluster_region = var.aks_cluster_region
    })
  ]

  # this is an override example
  set {
    name  = "components.castai-agent.overrides.additionalEnv.EXTRA_FLAG"
    value = "example_flag"
  }

  # this is a must, it needs to be dependant on operator being up and running
  depends_on = [
    helm_release.castware_operator
  ]
}
