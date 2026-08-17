---
page_title: "V1 to V2 Autoscaler and Evictor Migration Guide"
subcategory: ""
description: |-
  Migrate from castai_autoscaler (V1) to castai_autoscaler_policies and castai_evictor (V2).
---

# V1 to V2 Autoscaler and Evictor Migration Guide

The `castai_autoscaler` resource (V1) is **deprecated**. Use the V2 resources `castai_autoscaler_policies` and `castai_evictor` instead. The V1 resource will remain functional for this release but will be removed in the next major version of the provider.

## Overview

V2 splits cluster autoscaling configuration into two focused resources:

- `castai_autoscaler_policies` — manages the V2 cluster autoscaler policies (cluster limits, node downscaler, unschedulable pods).
- `castai_evictor` — manages the standalone evictor configuration via the V2 workload-eviction API.

`castai_evictor_advanced_config` continues to exist and has been migrated to the V2 workload-eviction API. Its schema is unchanged, so existing configurations continue to work.

## Deprecation timeline

- **Now** — `castai_autoscaler` (V1) is deprecated. A deprecation warning is shown on plan and apply.
- **Next major version** — the `castai_autoscaler` resource will be removed entirely.

This release is a **minor version bump**. The provider version is injected at build time by GoReleaser; release the new version by creating a git tag such as `v1.X.0`. Do not edit a version file.

## Important: no automatic translation

Internal ticket **CO-4291** (automatic V1-to-V2 translation in the backend) was marked as **Won't Do**. The CAST AI API will **not** automatically convert a V1 autoscaler configuration into a V2 configuration. You must migrate your Terraform configuration manually using the mapping below.

## V1 to V2 field mapping

### Policies

| V1 `castai_autoscaler` field path | V2 `castai_autoscaler_policies` field path | Notes |
|---|---|---|
| `autoscaler_settings.enabled` | `enabled` | Global master switch. |
| `autoscaler_settings.is_scoped_mode` | `scoped_mode` | Renamed. |
| `autoscaler_settings.cluster_limits` | `cluster_limits` | Same structure; see CPU mapping below. |
| `autoscaler_settings.node_downscaler` | `node_downscaler` | Same structure; see empty nodes mapping below. |
| `autoscaler_settings.unschedulable_pods` | `unschedulable_pods` | Same structure. |
| `autoscaler_settings.unschedulable_pods.pod_pinner` | `unschedulable_pods.pod_pinner` | Same structure. |
| `autoscaler_settings.node_downscaler.evictor` | `castai_evictor.*` | Moved to a separate resource. |

#### CPU limits

| V1 `cluster_limits.cpu` field | V2 `cluster_limits.cpu` field | Notes |
|---|---|---|
| `max_cores` | `max_cores` | Required in V2. |
| `min_cores` | `min_cores` | Deprecated in V2; no longer enforced by the backend. |

#### Empty nodes downscaler

| V1 `node_downscaler.empty_nodes` field | V2 `node_downscaler` field | Notes |
|---|---|---|
| `enabled` | `empty_nodes_enabled` | Renamed. |
| `delay_seconds` | `empty_nodes_delay` | V1 used an integer number of seconds; V2 uses a Go duration string such as `"90s"`. |

### Evictor

All evictor settings from `castai_autoscaler.autoscaler_settings.node_downscaler.evictor` move to the top level of `castai_evictor`:

| V1 evictor field | V2 `castai_evictor` field | Notes |
|---|---|---|
| `enabled` | `enabled` |  |
| `dry_run` | `dry_run` |  |
| `aggressive_mode` | `aggressive_mode` |  |
| `scoped_mode` | `scoped_mode` |  |
| `cycle_interval` | `cycle_interval` | Go duration string. |
| `node_grace_period_minutes` | `node_grace_period_minutes` |  |
| `pod_eviction_failure_back_off_interval` | `pod_eviction_failure_back_off_interval` | Go duration string. |
| `ignore_pod_disruption_budgets` | `ignore_pod_disruption_budgets` |  |
| `soft_tainting` | `soft_tainting` |  |
| `emit_node_related_pod_events` | `emit_node_related_pod_events` |  |
| `drain_timeout` | `drain_timeout` | New in V2; Go duration string. May not be active in the backend yet. |
| `drain_rollback_timeout` | `drain_rollback_timeout` | New in V2; Go duration string. May not be active in the backend yet. |

V2 also exposes 13 additional forward-compatible fields. See the [`castai_evictor` resource documentation](../resources/evictor.md) for the full list and descriptions.

## Dropped fields

The following V1 `castai_autoscaler` fields have no equivalent in `castai_autoscaler_policies` (V2). If you depend on any of these fields, continue using `castai_autoscaler` (V1) until they are added to the V2 resources.

- `node_templates_partial_matching_enabled` — No V2 equivalent yet. If you relied on this, keep using `castai_autoscaler` until the field is added to `castai_autoscaler_policies`.
- `node_downscaler.enabled` — The V2 `node_downscaler` block only exposes `empty_nodes_enabled` and `empty_nodes_delay`. There is no master toggle for the downscaler itself.
- `spot_instances` (and nested fields) — Deprecated in V1. Manage spot instance settings via Node Templates instead.
- `unschedulable_pods.headroom` / `headroom_spot` — No V2 equivalent. Manage via Node Templates.
- `unschedulable_pods.node_constraints` — No V2 equivalent. Manage via Node Templates.
- `unschedulable_pods.custom_instances_enabled` — No V2 equivalent.

> **Note:** If you depend on any of these fields, continue using `castai_autoscaler` (V1) until they are added to the V2 resources.

## Step-by-step migration

1. Review the field mapping table above and identify the settings you currently use in `castai_autoscaler`.
2. Remove the `castai_autoscaler` resource block from your Terraform configuration.
3. Add a `castai_autoscaler_policies` resource for the policy settings.
4. Add a `castai_evictor` resource for the evictor settings.
5. Convert V1 values where needed:
   - Rename `is_scoped_mode` to `scoped_mode`.
   - Rename `node_downscaler.empty_nodes.enabled` to `node_downscaler.empty_nodes_enabled`.
   - Convert `node_downscaler.empty_nodes.delay_seconds` from an integer to a duration string (for example, `90` becomes `"90s"`).
   - Move all evictor fields out of `node_downscaler.evictor` and into the `castai_evictor` resource.
6. Run `terraform plan` and review the diff. You may need to remove the old `castai_autoscaler` state and import the new resources using the cluster ID.
7. Apply the migrated configuration.

## Example: before (V1) and after (V2)

### Before: V1 `castai_autoscaler`

```terraform
resource "castai_autoscaler" "castai_autoscaler_policy" {
  cluster_id = castai_eks_cluster.test.id

  autoscaler_settings {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true
    }

    cluster_limits {
      enabled = true

      cpu {
        min_cores = 1
        max_cores = 10
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled       = true
        delay_seconds = 90
      }

      evictor {
        enabled                                = true
        dry_run                                = false
        aggressive_mode                        = false
        scoped_mode                            = false
        cycle_interval                         = "60s"
        node_grace_period_minutes              = 10
        pod_eviction_failure_back_off_interval = "30s"
        ignore_pod_disruption_budgets          = false
      }
    }
  }
}
```

### After: V2 `castai_autoscaler_policies` + `castai_evictor`

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
    enabled = true

    pod_pinner {
      enabled = true
    }
  }
}

resource "castai_evictor" "evictor" {
  cluster_id = castai_eks_cluster.test.id

  enabled                                = true
  dry_run                                = false
  aggressive_mode                        = false
  scoped_mode                            = false
  cycle_interval                         = "60s"
  node_grace_period_minutes              = 10
  pod_eviction_failure_back_off_interval = "30s"
  ignore_pod_disruption_budgets          = false
}
```

## Resources

- [`castai_autoscaler_policies` resource documentation](../resources/autoscaler_policies.md)
- [`castai_evictor` resource documentation](../resources/evictor.md)
- [`castai_evictor_advanced_config` resource documentation](../resources/evictor_advanced_config.md)
