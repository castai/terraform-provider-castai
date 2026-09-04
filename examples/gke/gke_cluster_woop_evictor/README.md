# GKE cluster with WOOP, Evictor, and Live Migration

Creates a GKE cluster with CAST AI workload optimization, node consolidation, and live migration.

## What it deploys

| Component | How | Purpose |
|---|---|---|
| **WOOP** (workload autoscaler) | `install_workload_autoscaler = true` | Right-sizes CPU/memory requests based on usage metrics |
| **Evictor** | `castai_evictor` V2 resource with `aggressive_mode = true` | Consolidates pods onto fewer nodes so empty nodes can be removed |
| **Live Migration (CLM)** | `install_live = true`, `clm_enabled = true` on default template | Migrates workloads off draining nodes without downtime |
| **Pod Pinner** | `pod_pinner.enabled = true` under `unschedulable_pods` | Pins pods to nodes after scheduling decisions |
| **WOOP policy** | `castai_workload_scaling_policy.default` | Generic CPU/memory scaling policy applied to all workloads |
| **Rebalancing** | `castai_rebalancing_schedule` every 10 min, 5% savings threshold | Periodic cluster optimization |

### Key design decisions

- **Evictor uses V2 API** (`castai_evictor` resource) instead of the deprecated `autoscaler_settings.node_downscaler.evictor` block to avoid dual-write conflicts.
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
```
