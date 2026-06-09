resource "helm_release" "castai_agent" {
  count            = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace
  name             = "castai-agent"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-agent"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  version = var.agent_version
  values  = var.agent_values

  set = concat(
    [
      {
        name  = "replicaCount"
        value = "2"
      },
      {
        name  = "provider"
        value = "gke"
      },
      {
        name  = "additionalEnv.STATIC_CLUSTER_ID"
        value = castai_gke_cluster.castai_cluster[0].id
      },
      {
        name  = "createNamespace"
        value = "false"
      },
    ],
    var.castai_api_url != "" ? [{
      name  = "apiURL"
      value = var.castai_api_url
    }] : [],
    [for k, v in var.castai_components_labels : {
      name  = "podLabels.${k}"
      value = v
    }],
  )

  set_sensitive = [
    {
      name  = "apiKey"
      value = castai_gke_cluster.castai_cluster[0].cluster_token
    },
  ]
}

resource "helm_release" "castai_cluster_controller" {
  count = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace

  name             = "cluster-controller"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-cluster-controller"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  version = var.cluster_controller_version
  values  = var.cluster_controller_values

  set = concat(
    [
      {
        name  = "castai.clusterID"
        value = castai_gke_cluster.castai_cluster[0].id
      }
    ],
    var.castai_api_url != "" ? [{
      name  = "castai.apiURL"
      value = var.castai_api_url
    }] : [],
    [for k, v in var.castai_components_labels : {
      name  = "podLabels.${k}"
      value = v
    }],
  )

  set_sensitive = [
    {
      name  = "castai.apiKey"
      value = castai_gke_cluster.castai_cluster[0].cluster_token
    },
  ]

  depends_on = [helm_release.castai_agent]

  lifecycle {
    ignore_changes = [version]
  }
}

resource "helm_release" "castai_evictor" {
  count = terraform.workspace == var.org_workspace ? 0 : 1 # Create only in the cluster workspace

  name             = "castai-evictor"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-evictor"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true
  wait             = true

  version = var.evictor_version
  values  = var.evictor_values

  set = concat(
    [
      {
        name  = "replicaCount"
        value = "0"
      },
      {
        name  = "castai-evictor-ext.enabled"
        value = "false"
      },
    ],
    [for k, v in var.castai_components_labels : {
      name  = "podLabels.${k}"
      value = v
    }],
  )

  depends_on = [helm_release.castai_agent]

  lifecycle {
    ignore_changes = [set, version]
  }
}
