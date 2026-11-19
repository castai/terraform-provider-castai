# IAM resources required by the CAST AI Storage Autoscaler and by Kvisor storage metrics.
#
# Both components call GCP APIs from inside the cluster. Access is granted via
# Workload Identity Federation: every GCP service account below is federated
# with a dedicated Kubernetes service account.
#
#   castai-agent/castai-storage-autoscaler - expands disks, right-sizes IOPS/throughput
#   castai-agent/castai-kvisor-controller  - lists disks and instances for storage metrics
#
# The identities are intentionally separate, so that Kvisor does not inherit the
# disk update permissions of the Storage Autoscaler.
#
# See https://docs.cast.ai/docs/storage-installation and
# https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics for details.

locals {
  # Names are shared with castai.tf, so the Workload Identity bindings always match
  # the service accounts created by the CAST AI Helm chart.
  castai_namespace                   = "castai-agent"
  storage_autoscaler_service_account = "castai-storage-autoscaler"
  kvisor_controller_service_account  = "castai-kvisor-controller"
}

################################################################################
# Storage Autoscaler
################################################################################

resource "google_service_account" "castai_storage_autoscaler" {
  account_id   = "castai-storage-autoscaler"
  display_name = "CAST AI Storage Autoscaler controller"
}

# Read disk metadata and modify zonal and regional disk IOPS, throughput,
# and size via the GCP Compute API. Regional disk API calls use these same
# compute.disks.* permissions; compute.regionDisks.* names are not required.
resource "google_project_iam_custom_role" "castai_storage_autoscaler_volume" {
  project     = var.project_id
  role_id     = "castaiStorageAutoscalerVolume"
  title       = "CAST AI Storage Autoscaler volume manager"
  description = "Allows Storage Autoscaler to read, resize, and modify zonal and regional persistent disks"
  permissions = [
    "compute.disks.get",    # Read current disk state and provisioned performance.
    "compute.disks.list",   # Discover zonal and regional disks.
    "compute.disks.update", # Update provisioned IOPS and throughput.
    "compute.disks.resize", # Expand zonal and regional disk size.
  ]
}

resource "google_project_iam_member" "castai_storage_autoscaler_volume" {
  project = var.project_id
  role    = google_project_iam_custom_role.castai_storage_autoscaler_volume.id
  member  = "serviceAccount:${google_service_account.castai_storage_autoscaler.email}"
}

# Fetch actual disk IOPS/throughput consumption from GCP Cloud Monitoring.
resource "google_project_iam_member" "castai_storage_autoscaler_monitoring_viewer" {
  project = var.project_id
  role    = "roles/monitoring.viewer"
  member  = "serviceAccount:${google_service_account.castai_storage_autoscaler.email}"
}

# Allows the `castai-storage-autoscaler` Kubernetes service account to act as the
# GCP service account above (Workload Identity Federation).
resource "google_service_account_iam_member" "castai_storage_autoscaler_workload_identity" {
  service_account_id = google_service_account.castai_storage_autoscaler.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${local.castai_namespace}/${local.storage_autoscaler_service_account}]"
}

################################################################################
# Kvisor storage metrics
################################################################################

resource "google_service_account" "castai_kvisor" {
  account_id   = "castai-kvisor"
  display_name = "CAST AI Kvisor controller"
}

# Kvisor controller collects cloud volume metadata by listing disks and instances.
# A custom role grants exactly these permissions; the predefined roles/compute.viewer
# would also work, but gives read access to all Compute Engine resources.
#
# NOTE: GCP soft-deletes custom roles. After `terraform destroy` the role_id can't be
# reused for up to 37 days - change role_id if re-applying fails because of that.
resource "google_project_iam_custom_role" "castai_kvisor_storage_reader" {
  project     = var.project_id
  role_id     = "castaiKvisorStorageReader"
  title       = "CAST AI Kvisor storage reader"
  description = "Allows Kvisor to list disks and instances to collect cloud volume metrics"
  permissions = [
    "compute.disks.list",
    "compute.instances.list",
  ]
}

resource "google_project_iam_member" "castai_kvisor_storage_reader" {
  project = var.project_id
  role    = google_project_iam_custom_role.castai_kvisor_storage_reader.id
  member  = "serviceAccount:${google_service_account.castai_kvisor.email}"
}

# Allows the `castai-kvisor-controller` Kubernetes service account to act as the
# GCP service account above (Workload Identity Federation).
resource "google_service_account_iam_member" "castai_kvisor_workload_identity" {
  service_account_id = google_service_account.castai_kvisor.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${local.castai_namespace}/${local.kvisor_controller_service_account}]"
}
