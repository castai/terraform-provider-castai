# GKE cluster with WOOP, Evictor, and Live Migration (umbrella Helm chart)

Creates a GKE cluster with CAST AI workload optimization, node consolidation, and live migration.

## Architecture

This example uses the **CAST AI umbrella Helm chart** (`castai-helm/castai`) with
`tags.full = true` instead of the `castai/gke-cluster` module for Helm releases.

| Layer | Managed by | What |
|---|---|---|
| **Infrastructure** | Terraform | VPC, GKE cluster, GCP IAM (`castai/gke-iam`) |
| **Cluster registration** | Terraform | `castai_gke_cluster` resource (returns `cluster_id` + `cluster_token`) |
| **Helm release** | Terraform (`helm_release`) | Single umbrella chart `castai` with `tags.full = true` |
| **CAST AI config** | Terraform | Node configurations, node templates, autoscaler settings, evictor (V2), WOOP policy, rebalancing schedule |

CAST AI does **not** auto-upgrade the umbrella chart — so there is no version
conflict between CAST AI's auto-updater and the Terraform-managed `helm_release`.

### What it deploys

| Component | How | Purpose |
|---|---|---|
| **Umbrella chart** | `helm_release "castai_umbrella"` with `tags.full = true` | agent, cluster-controller, evictor, pod-mutator, pod-pinner, live, workload-autoscaler, spot-handler, kvisor, workload-autoscaler-exporter |
| **Evictor** | `castai_evictor` V2 resource with `aggressive_mode = true` | Consolidates pods onto fewer nodes so empty nodes can be removed |
| **Live Migration (CLM)** | `clm_enabled = true` on default template + init scripts | Migrates workloads off draining nodes without downtime |
| **Pod Pinner** | `pod_pinner.enabled = true` under `unschedulable_pods` | Pins pods to nodes after scheduling decisions |
| **WOOP policy** | `castai_workload_scaling_policy.default` | Generic CPU/memory scaling policy applied to all workloads |
| **Rebalancing** | `castai_rebalancing_schedule` every 10 min, 5% savings threshold | Periodic cluster optimization |

### Key design decisions

- **Umbrella chart instead of the `castai/gke-cluster` module** — Terraform installs a single `castai` Helm release with `tags.full = true`. The `castai_gke_cluster` resource is a direct resource (not a module) that registers the cluster and returns `cluster_token`.
- **Evictor uses V2 API** (`castai_evictor` resource) instead of the deprecated `autoscaler_settings.node_downscaler.evictor` block to avoid dual-write conflicts.
- **No evictor block in autoscaler_settings** — the `node_downscaler` block only configures `empty_nodes`; evictor config comes solely from `castai_evictor`.
- **Single node config and template** — Live Migration is enabled on the `default_by_castai` template via `clm_enabled = true`, no separate `live` config needed.
- **CLM init scripts** — `init_cos.sh` / `init_ubuntu.sh` install `cri-proxy` on nodes, required for Live Migration.

## How to use

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in your values.
2. `terraform init && terraform apply`
3. `terraform destroy` when done.

## Verify after apply

```bash
# Evictor config synced to cluster
kubectl get evictorconfig evictor-config -o yaml | grep aggressiveMode
# Should show: aggressiveMode: true

# Evictor status from Terraform
terraform output evictor_status
# Should show: Compatible

# Cluster ID and token
terraform output cluster_id
terraform output -raw cluster_token
```
