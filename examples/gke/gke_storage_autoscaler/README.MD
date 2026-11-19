## Existing GKE cluster with CAST AI Storage Autoscaling and Kvisor storage metrics

Following example shows how to onboard an existing GKE cluster to CAST AI and enable [Storage Autoscaling](https://docs.cast.ai/docs/storage-introduction) together with [Kvisor storage metrics](https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics).

Storage Autoscaling expands PVCs and right-sizes Persistent Disk IOPS/throughput. Kvisor feeds the Storage autoscaler dashboard in the Cast AI console (savings estimates, storage type breakdown, resize event logs, per-PV usage). Storage Autoscaling still works without Kvisor, but the dashboard stays empty.

Both components call GCP APIs from inside the cluster, so they need dedicated GCP service accounts, which are **not** created by the `castai/gke-cluster` module. This example provisions them in `iam.tf` as two separate identities, so that Kvisor does not inherit the write permissions of the Storage Autoscaler.

**Storage Autoscaler** (`castai-agent/castai-storage-autoscaler` service account), following the [manual GKE IAM setup](https://docs.cast.ai/docs/storage-installation#gke-identity-and-permissions-manual-setup):

| IAM role or permission | Purpose |
| --- | --- |
| `compute.disks.get` | Read current disk state and provisioned performance |
| `compute.disks.list` | Discover zonal and regional disks |
| `compute.disks.update` | Update provisioned IOPS and throughput |
| `compute.disks.resize` | Expand zonal and regional disk size |
| `roles/monitoring.viewer` | Fetch actual disk IOPS/throughput consumption from GCP Cloud Monitoring |

The disk permissions are granted through a project-level custom role instead of the broad `roles/compute.storageAdmin`. Regional `regionDisks` API calls use the same `compute.disks.*` permissions; corresponding `compute.regionDisks.*` permission names are not required.

**Kvisor controller** (`castai-agent/castai-kvisor-controller` service account), following the [Kvisor storage metrics guide](https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics):

| IAM permission | Purpose |
| --- | --- |
| `compute.disks.list` | List persistent disks to collect volume metadata |
| `compute.instances.list` | List instances to map disks to nodes |

The two permissions are granted through a project-level custom role, so Kvisor gets nothing beyond them. The predefined `roles/compute.viewer` role would also work, but gives read access to all Compute Engine resources.

Each GCP service account is federated with the matching Kubernetes service account via [Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) (`roles/iam.workloadIdentityUser`). The service account emails are passed to the `castai/gke-cluster` module via `storage_autoscaler_values` and `kvisor_values`, which annotate the Kubernetes service accounts created by the umbrella Helm chart. `kvisor_values` also turns on storage metrics collection: `storage-stats-enabled` on the Kvisor agent and `cloud-provider-storage-sync-enabled` on the Kvisor controller.

IAM resources required to connect the cluster itself to CAST AI are created by the [castai/gke-iam/castai module](https://github.com/castai/terraform-castai-gke-iam).

Example configuration should be analysed in the following order:
1. Creates the GCP service accounts, IAM role bindings, and Workload Identity Federation bindings for the Storage Autoscaler and for Kvisor storage metrics - `iam.tf`
2. Creates CAST AI resources to connect the GKE cluster to CAST AI, and enables Storage Autoscaling and Kvisor - `castai.tf`

# Prerequisites
- GKE cluster running Kubernetes 1.29 or later with [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) enabled (verified by a `check` block during `terraform plan`)
- A StorageClass with `allowVolumeExpansion: true`
- Permission to create custom IAM roles in the project (`iam.roles.create`)

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your project id, cluster name, cluster location and CAST AI API token.

| Variable | Description |
| --- | --- |
| cluster_name     = "" | Name of cluster |
| cluster_location = "" | Location of the cluster (region or zone) |
| castai_api_token = "" | Cast api token |
| project_id       = "" | GCP project ID |

3. Initialize Terraform. Under example root folder run:
```
terraform init
```
4. Run Terraform apply:
```
terraform apply
```
5. To destroy resources created by this example:
```
terraform destroy
```

> **Note**
>
> - Storage Autoscaling installs in **dry-run mode**: planned actions are logged without modifying PVCs. Review the dry-run logs (`kubectl logs -n castai-agent -l app.kubernetes.io/name=castai-storage-autoscaler --follow`), then enable live mode as described in [Enable live mode](https://docs.cast.ai/docs/storage-installation#enable-live-mode) - e.g. flip `config.dryRun` to `false` in `storage_autoscaler_values` or patch the default PVPolicy.
> - The `check` block verifies Workload Identity at the cluster level only. The nodes running the Storage Autoscaler and Kvisor controller pods must also run the GKE metadata server (`GKE_METADATA` node pool mode), otherwise the pods cannot obtain the service account credentials.
> - GCP soft-deletes custom roles. After `terraform destroy`, the `castaiKvisorStorageReader` and `castaiStorageAutoscalerVolume` role IDs cannot be reused for up to 37 days. Change the affected `role_id` in `iam.tf` if re-applying fails because of that.
> - `workload_autoscaler_keep_crds=true` is required with `umbrella_enabled=true`. For a new onboarding, as assumed by this example, no separate migration is needed. When converting an installation that already has standalone CAST AI Helm charts, first apply the standalone configuration with `umbrella_enabled=false` and `workload_autoscaler_keep_crds=true`, using the compatible module version from the [umbrella migration guide](https://github.com/castai/terraform-provider-castai/tree/master/docs/umbrella-migration). Only then enable the umbrella chart in a separate apply.
> - `kvisor_grpc_addr` controls Kvisor's separate `CASTAI_API_GRPC_ADDR` and is independent of `castai_grpc_url`. Use `kvisor.prod-eu.cast.ai:443` for the EU console (host:port only, without `https://`). The main API URL is separately inherited by Kvisor from `global.castai.apiURL`.
> - The `install_storage_autoscaler` and `storage_autoscaler_values` inputs require `castai/gke-cluster` module v12.0.0 or newer ([storage autoscaler support](https://github.com/castai/terraform-castai-gke-cluster/pull/160)).

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
