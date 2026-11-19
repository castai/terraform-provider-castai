# Configure Data sources and providers required for CAST AI connection.
data "google_client_config" "default" {}

data "google_container_cluster" "my_cluster" {
  name     = var.cluster_name
  location = var.cluster_location
  project  = var.project_id
}

# Storage Autoscaling and Kvisor storage metrics use Workload Identity Federation to call
# GCP APIs. The cluster must have Workload Identity enabled.
check "workload_identity_enabled" {
  assert {
    condition     = try(data.google_container_cluster.my_cluster.workload_identity_config[0].workload_pool, "") != ""
    error_message = "GKE cluster ${var.cluster_name} does not have Workload Identity enabled. Storage Autoscaling and Kvisor storage metrics require it - see https://docs.cast.ai/docs/storage-installation"
  }
}

# Configure GKE cluster connection using CAST AI gke-cluster module.
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.5"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}

module "castai_gke_cluster" {
  source  = "castai/gke-cluster/castai"
  version = "~> 12.0" # Storage Autoscaler support (install_storage_autoscaler) was added in v12.0.0

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  kvisor_grpc_addr       = var.kvisor_grpc_addr
  wait_for_cluster_ready = true

  # Required for umbrella installations. Keeps Workload Autoscaler CRDs if standalone
  # charts are removed in favor of the umbrella chart. Existing standalone installations
  # must apply this with umbrella_enabled=false first, then enable the umbrella separately.
  workload_autoscaler_keep_crds = true
  umbrella_enabled              = true

  project_id           = var.project_id
  gke_cluster_name     = var.cluster_name
  gke_cluster_location = var.cluster_location

  gke_credentials            = module.castai_gke_iam.private_key
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai_gke_cluster.castai_node_configurations["default"]
  node_configurations = {
    default = {
      min_disk_size  = 100
      disk_cpu_ratio = 0
      subnets        = [data.google_container_cluster.my_cluster.subnetwork]
      tags           = var.tags
    }
  }

  # Installs Storage Autoscaler sub-chart of the Cast AI umbrella Helm chart.
  # Installation starts in dry-run mode: planned actions are logged, no PVCs are modified.
  install_storage_autoscaler = true

  # Binds the GCP service account from iam.tf to the `castai-storage-autoscaler`
  # Kubernetes service account so the controller can call GCP APIs.
  storage_autoscaler_values = [
    yamlencode({
      config = {
        # Set to false to let Storage Autoscaler apply changes to PVCs (live mode).
        # See https://docs.cast.ai/docs/storage-installation#enable-live-mode
        dryRun = true
      }
      serviceAccount = {
        name = local.storage_autoscaler_service_account
        annotations = {
          "iam.gke.io/gcp-service-account" = google_service_account.castai_storage_autoscaler.email
        }
      }
    })
  ]

  # Installs Kvisor (https://docs.cast.ai/docs/kvisor). Its storage metrics populate the
  # Storage Autoscaler dashboard in the Cast AI console: savings estimates, storage type
  # breakdown, resize event logs and per-PV usage. Storage Autoscaler works without Kvisor,
  # but the dashboard stays empty.
  install_security_agent = true

  # See https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics
  kvisor_values = [
    yamlencode({
      controller = {
        # Storage metrics collection requires both the Kvisor controller and agent.
        enabled = true

        extraArgs = {
          # Collect cloud volume metadata (disk type, size, provisioned IOPS) from the GCP Compute API.
          "cloud-provider"                      = "gcp"
          "cloud-provider-gcp-project-id"       = var.project_id
          "cloud-provider-storage-sync-enabled" = true
        }

        # Binds the GCP service account from iam.tf (compute.disks.list, compute.instances.list)
        # to the Kvisor controller.
        serviceAccount = {
          name = local.kvisor_controller_service_account
          annotations = {
            "iam.gke.io/gcp-service-account" = google_service_account.castai_kvisor.email
          }
        }
      }

      agent = {
        enabled = true

        extraArgs = {
          # Collect ephemeral storage and persistent volume usage per node.
          "storage-stats-enabled" = true
        }
      }
    })
  ]

  // depends_on helps terraform with creating proper dependencies graph in case of resource creation and in this case destroy
  // module "castai-gke-cluster" has to be destroyed before module "castai-gke-iam"
  depends_on = [data.google_container_cluster.my_cluster, module.castai_gke_iam]
}
