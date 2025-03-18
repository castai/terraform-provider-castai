# Start Minikube using the Docker driver
resource "null_resource" "start_minikube" {
  provisioner "local-exec" {
    command = "minikube start --driver=docker"
  }
}

# Wait for Minikube to be ready
data "external" "wait_for_minikube" {
  program = ["bash", "-c", "while ! kubectl get nodes >/dev/null 2>&1; do sleep 5; done; echo '{}'"]
  depends_on = [null_resource.start_minikube]
}

# Configure the Kubernetes provider
provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "minikube"
}

# Configure the Helm provider
provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "minikube"
  }
}

# Create the namespace "castai-agent"
resource "kubernetes_namespace" "castai" {
  depends_on = [data.external.wait_for_minikube]
  metadata {
    name = "castai-agent"
    labels = {
      "app.kubernetes.io/managed-by" = "Helm"
    }
    annotations = {
      "meta.helm.sh/release-name"      = "castai-agent"
      "meta.helm.sh/release-namespace" = "castai-agent"
    }
  }
}

# Install CAST AI Agent
resource "helm_release" "castai_agent" {
  depends_on       = [kubernetes_namespace.castai]
  name             = "castai-agent"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-agent"
  namespace        = "castai-agent"
  create_namespace = false
  timeout          = 600  

  set {
    name  = "apiKey"
    value = var.cast_ai_api_key
  }
  set {
    name  = "clusterName"
    value = var.cluster_name
  }
  set {
    name  = "provider"
    value = "anywhere"
  }
}

# Install CAST AI Cluster Controller
resource "helm_release" "castai_cluster_controller" {
  depends_on       = [helm_release.castai_agent]
  name             = "castai-cluster-controller"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-cluster-controller"
  namespace        = "castai-agent"
  create_namespace = false
  timeout          = 600  

  set {
    name  = "castai.apiKey"
    value = var.cast_ai_api_key
  }
  set {
    name  = "castai.clusterID"
    value = var.cluster_id
  }
  set {
    name  = "enableTopologySpreadConstraints"
    value = "true"
  }
}

# Install CAST AI Evictor
resource "helm_release" "castai_evictor" {
  depends_on       = [helm_release.castai_cluster_controller]
  name             = "castai-evictor"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-evictor"
  namespace        = "castai-agent"
  create_namespace = false
  timeout          = 600  

  set {
    name  = "managedByCASTAI"
    value = var.managed_by_castai
  }
  set {
    name  = "replicaCount"
    value = "1"
  }
    set {
    name  = "aggressive_mode"
    value = "true"
  }
}

# Install CAST AI Pod Mutator
resource "helm_release" "castai_pod_mutator" {
  depends_on       = [helm_release.castai_cluster_controller]
  name             = "castai-pod-mutator"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-pod-mutator"
  namespace        = "castai-agent"
  create_namespace = false
  timeout          = 600  

  set {
    name  = "castai.apiKey"
    value = var.cast_ai_api_key
  }
  set {
    name  = "castai.clusterID"
    value = var.cluster_id
  }
  set {
    name  = "enableTopologySpreadConstraints"
    value = "true"
  }

    set {
    name  = "castai.organizationID"
    value = var.organization_id  # <-- Add this line
  }
}



# Install CAST AI Workload Autoscaler
resource "helm_release" "castai_workload_autoscaler" {
  depends_on       = [helm_release.castai_pod_mutator]
  name             = "castai-workload-autoscaler"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-workload-autoscaler"
  namespace        = "castai-agent"
  create_namespace = false
  timeout          = 600  

  set {
    name  = "castai.apiKey"
    value = var.cast_ai_api_key
  }
  set {
    name  = "castai.clusterID"
    value = var.cluster_id
  }
}
