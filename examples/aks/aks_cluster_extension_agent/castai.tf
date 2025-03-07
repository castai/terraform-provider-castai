# 5. Connect cluster to Cast AI and deploy Castware components.

data "azurerm_subscription" "current" {}

resource "castai_aks_cluster" "castai_cluster" {
  name = var.cluster_name

  region          = var.cluster_region
  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id
  client_id       = azuread_application.castai.client_id
  client_secret   = azuread_application_password.castai.value

  node_resource_group        = azurerm_kubernetes_cluster.this.node_resource_group
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  # CastAI needs cloud permission to do some clean up
  # when disconnecting the cluster.
  # This ensures IAM configurations exist during disconnect.
  depends_on = [
    azurerm_role_definition.castai,
    azurerm_role_assignment.castai_resource_group,
    azurerm_role_assignment.castai_node_resource_group,
    azurerm_role_assignment.castai_additional_resource_groups,
    azuread_application.castai,
    azuread_application_password.castai,
    azuread_service_principal.castai
  ]
}

resource "helm_release" "castai_cluster_controller" {

  name             = "cluster-controller"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-cluster-controller"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "aks.enabled"
    value = "true"
  }

  set {
    name  = "castai.clusterID"
    value = castai_aks_cluster.castai_cluster.id
  }

  dynamic "set" {
    for_each = var.castai_api_url != "" ? [var.castai_api_url] : []
    content {
      name  = "castai.apiURL"
      value = var.castai_api_url
    }
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_aks_cluster.castai_cluster.cluster_token
  }

  depends_on = [azurerm_kubernetes_cluster_extension.castai]
}

resource "helm_release" "castai_evictor" {

  name             = "castai-evictor"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-evictor"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "castai-evictor-ext.enabled"
    value = "false"
  }

  depends_on = [azurerm_kubernetes_cluster_extension.castai]
}

resource "helm_release" "castai_pod_pinner" {
  name             = "castai-pod-pinner"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-pod-pinner"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "castai.clusterID"
    value = castai_aks_cluster.castai_cluster.id
  }

  dynamic "set" {
    for_each = var.castai_api_url != "" ? [var.castai_api_url] : []
    content {
      name  = "castai.apiURL"
      value = var.castai_api_url
    }
  }

  dynamic "set" {
    for_each = var.grpc_url != "" ? [var.grpc_url] : []
    content {
      name  = "castai.grpcURL"
      value = var.grpc_url
    }
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_aks_cluster.castai_cluster.cluster_token
  }

  set {
    name  = "replicaCount"
    value = "0"
  }

  depends_on = [azurerm_kubernetes_cluster_extension.castai]

  lifecycle {
    ignore_changes = [version]
  }
}

resource "helm_release" "castai_spot_handler" {
  name             = "castai-spot-handler"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-spot-handler"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "castai.provider"
    value = "azure"
  }

  set {
    name  = "createNamespace"
    value = "false"
  }

  dynamic "set" {
    for_each = var.castai_api_url != "" ? [var.castai_api_url] : []
    content {
      name  = "castai.apiURL"
      value = var.castai_api_url
    }
  }

  set {
    name  = "castai.clusterID"
    value = castai_aks_cluster.castai_cluster.id
  }

  depends_on = [azurerm_kubernetes_cluster_extension.castai]
}

resource "helm_release" "castai_workload_autoscaler" {
  name             = "castai-workload-autoscaler"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-workload-autoscaler"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  set {
    name  = "castai.apiKeySecretRef"
    value = "castai-cluster-controller"
  }

  set {
    name  = "castai.configMapRef"
    value = "castai-cluster-controller"
  }

  depends_on = [azurerm_kubernetes_cluster_extension.castai, helm_release.castai_cluster_controller]

}


resource "helm_release" "castai_kvisor" {

  name             = "castai-kvisor"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-kvisor"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true


  lifecycle {
    ignore_changes = [version]
  }

  set {
    name  = "castai.clusterID"
    value = castai_aks_cluster.castai_cluster.id
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_aks_cluster.castai_cluster.cluster_token
  }

  set {
    name  = "castai.grpcAddr"
    value = var.kvisor_grpc_url
  }

  set {
    name  = "controller.extraArgs.kube-bench-cloud-provider"
    value = "aks"
  }
}
