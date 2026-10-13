---
page_title: "castai_autoscaler_policies Resource - terraform-provider-castai"
subcategory: ""
description: |-
  CAST AI autoscaler policies V2 resource to manage cluster autoscaling policies.
---

# castai_autoscaler_policies (Resource)

CAST AI autoscaler policies V2 resource to manage cluster autoscaling policies.

## Example Usage

```terraform
resource "castai_autoscaler_policies" "policies" {
  cluster_id  = castai_eks_cluster.test.id
  enabled     = true
  scoped_mode = false

  cluster_limits {
    enabled = true

    cpu {
      max_cores = 10
    }
  }

  node_downscaler {
    empty_nodes_enabled = true
    empty_nodes_delay   = "90s"
  }

  unschedulable_pods {
    enabled                           = true
    partial_template_matching_enabled = true

    pod_pinner {
      enabled = true
    }
  }
}
```

## Migrating from `castai_autoscaler` (V1)

The V1 `castai_autoscaler` resource is deprecated. Use `castai_autoscaler_policies` together with [`castai_evictor`](evictor.md) instead.

### V1 → V2 field mapping

| V1 `castai_autoscaler` | V2 `castai_autoscaler_policies` | Notes |
|---|---|---|
| `enabled` | `enabled` | Same |
| `is_scoped_mode` | `scoped_mode` | Renamed |
| `cluster_limits` block | `cluster_limits` block | Same structure |
| `node_downscaler` block | `node_downscaler` block | Simplified — `empty_nodes` flattened to `empty_nodes_enabled` + `empty_nodes_delay` |
| `node_downscaler.evictor` | **Removed** | Now managed by `castai_evictor` resource |
| `unschedulable_pods` block | `unschedulable_pods` block | Same structure |
| `unschedulable_pods.pod_pinner` | `unschedulable_pods.pod_pinner` | Same |
| `node_templates_partial_matching_enabled` | `unschedulable_pods.partial_template_matching_enabled` | Moved inside `unschedulable_pods` |
| `spot_instances` | **Removed** | Manage via `castai_node_template` |
| `headroom` / `headroom_spot` | **Removed** | Deprecated in V1; manage via Node Templates |
| `node_constraints` | **Removed** | Manage via `castai_node_template` |
| `custom_instances_enabled` | **Removed** | Manage via `castai_node_template` |

### Dropped fields

The following V1 fields have no V2 equivalent. If you depend on them, continue using `castai_autoscaler` (V1) until they are added to the V2 resources:

- `spot_instances` (all nested fields) — manage via Node Templates
- `unschedulable_pods.headroom` / `headroom_spot` — manage via Node Templates
- `unschedulable_pods.node_constraints` — manage via Node Templates
- `unschedulable_pods.custom_instances_enabled` — manage via Node Templates

### Migration steps

1. Remove the `castai_autoscaler` resource block from your Terraform configuration.
2. Add `castai_autoscaler_policies` and `castai_evictor` resource blocks with the migrated fields.
3. Run `terraform state rm castai_autoscaler.<name>` to remove the old resource from state.
4. Run `terraform import castai_autoscaler_policies.<name> <cluster_id>` and `terraform import castai_evictor.<name> <cluster_id>`.
5. Run `terraform plan` to verify the new resources match the existing cluster state.

> **Warning:** V1 and V2 manage independent API stores. Do not run `castai_autoscaler` alongside `castai_autoscaler_policies` / `castai_evictor` on the same cluster.

### CO-4291 note

V1→V2 internal translation (CO-4291) was marked **Won't Do** — the V1 API will NOT auto-translate to V2. Users must manually migrate.

### Section and nested-block semantics

The server returns fully-materialized policies on every response and, for nested objects like `cluster_limits.cpu` and `node_downscaler.empty_nodes_delay`, applies partial-update merge on omitted fields. Leaf fields and `cluster_limits.enabled`, `partial_template_matching_enabled`, `pod_pinner.enabled`, `node_downscaler.empty_nodes_enabled` are not presence-merged — the server treats omission as the zero value.

- **Sections and nested blocks in the plan are the only ones tracked in state.** If you declare `cluster_limits` but omit `cluster_limits.cpu`, the plan has zero `cpu` entries and so does state; the server keeps its stored `cpu` but the resource does not manage it. Removing the block from your config stops managing it; re-adding the block re-binds it on the next apply.
- **Inner fields with a default mirror the server's defaults** so create-time plans agree with the server's first response. `min_cores` defaults to `1`, `empty_nodes_delay` defaults to `"300s"` (5 minutes), `empty_nodes_enabled` and the plain section bools default to `false`. Removing a default value from your config does not "go back to the server default" — it sets the value to the default itself, which the server sees as an explicit setting. To revert to the server default, leave the value unchanged from your last explicit setting, or open a ticket asking for presence-aware handling.
- **`cluster_limits.cpu.max_cores` is optional with a lower-bound check** (`≥ 2`). Omitting it sends no value, which the SDK cannot represent for this field (it's a non-pointer `int32` in the generated SDK); refresh adopts the server's stored value. Once the upstream API exposes `max_cores` as a pointer, this provider's regeneration will pick it up automatically.
- **All other inner fields are optional.** The SDK's pointer types (`*int32` for `min_cores`, `*string` for `empty_nodes_delay`, `*bool` for the section bools) let the provider omit them from the PUT request, which the server merges with the stored value. If the server's response carries the stored value back, the refresh helper preserves the plan's null so the next plan stays clean.
- **After `terraform import`, a section left out of the config shows once as a removal in the plan.** Applying it stops managing the section; re-declaring it re-binds it.
- **Upgrading from a provider version that held server-materialized sections in state** (any version before the `refactor(autoscaler_policies): declare inner-field defaults in schema` change in this provider): legacy state holding sections the config omits is the same shape as the "section removed from config" scenario. The first plan shows one removal; the next apply converges.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster_id` (String) CAST AI cluster id.

### Optional

- `cluster_limits` (Block List) Defines minimum and maximum amount of CPU the cluster can have. cluster_limits { enabled = true, cpu { max_cores = 100, min_cores = 1 } }. (see [below for nested schema](#nestedblock--cluster_limits))
- `enabled` (Boolean) Enable/disable all policies (global master switch).
- `node_downscaler` (Block List) Node Downscaler defines policies for removing nodes based on the configured conditions. node_downscaler { empty_nodes_enabled = true, empty_nodes_delay = "5m" }. (see [below for nested schema](#nestedblock--node_downscaler))
- `scoped_mode` (Boolean) Run the node autoscaler in scoped mode.
- `unschedulable_pods` (Block List) Policy defining autoscaler's behavior when unschedulable pods were detected. unschedulable_pods { enabled = true, partial_template_matching_enabled = false, pod_pinner { enabled = true } }. (see [below for nested schema](#nestedblock--unschedulable_pods))

### Read-Only

- `id` (String) The ID of this resource, equal to the cluster id.
- `version` (String) Policy version for optimistic locking.

<a id="nestedblock--cluster_limits"></a>
### Nested Schema for `cluster_limits`

Optional:

- `cpu` (Block List) (see [below for nested schema](#nestedblock--cluster_limits--cpu))
- `enabled` (Boolean) Enable/disable the cluster_limits policy.

<a id="nestedblock--cluster_limits--cpu"></a>
### Nested Schema for `cluster_limits.cpu`

Optional:

- `max_cores` (Number) Maximum vCPUs allowed cluster-wide.
- `min_cores` (Number) Minimum vCPUs allowed cluster-wide. Matches the server default when omitted.



<a id="nestedblock--node_downscaler"></a>
### Nested Schema for `node_downscaler`

Optional:

- `empty_nodes_delay` (String) How long a node must be empty before it becomes eligible for downscaling, as a duration in seconds (e.g. "300s" for 5 minutes). Matches the server default when omitted.
- `empty_nodes_enabled` (Boolean) Enable downscaling of empty nodes. The server treats absence as false; matching that here keeps plan and apply result in agreement.


<a id="nestedblock--unschedulable_pods"></a>
### Nested Schema for `unschedulable_pods`

Optional:

- `enabled` (Boolean) Enable the unschedulable pods policy.
- `partial_template_matching_enabled` (Boolean) Use partial template matching when deciding which custom node template to select.
- `pod_pinner` (Block List) Pod Pinner component settings. (see [below for nested schema](#nestedblock--unschedulable_pods--pod_pinner))

<a id="nestedblock--unschedulable_pods--pod_pinner"></a>
### Nested Schema for `unschedulable_pods.pod_pinner`

Optional:

- `enabled` (Boolean) Enable the Pod Pinner component.
