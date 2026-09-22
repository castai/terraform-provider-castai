# Parity Gap Analysis: SDK vs Terraform Schemas (Autoscaler Parity)

## Executive summary (consolidated)

Field-by-field parity comparison between the SDK regenerated from live `api.cast.ai` specs (2026-09-22, commit `360ecff`) and the Terraform provider schemas, across the three in-scope areas. **~189 request-body fields compared; 12 GAPs, 5 MISMATCHes found.** The v2 policies surface (`castai_autoscaler_policies`) is at full field coverage; all remaining gaps are in `castai_node_template`, `castai_rebalancing_schedule`, and the deprecated `castai_autoscaler` (V1). The new regen surface (`ExternalclusterV1DrainConfig.Pods`) is not mapped by any in-scope resource and needs no action.

| Area | Resource | Fields compared | GAP | MISMATCH |
| --- | --- | --- | --- | --- |
| Policies (V1, deprecated) | castai_autoscaler | ~47 | 4 | 1 |
| Policies (V2) | castai_autoscaler_policies | 17 | 0 | 2 |
| Node templates | castai_node_template | ~84 | 5 | 1 |
| Rebalancing | castai_rebalancing_schedule | ~34 | 3 | 1 |
| Rebalancing | castai_rebalancing_job | ~7 | 0 | 0 |
| **Total** | | **~189** | **12** | **5** |

### Status of gaps (post Phase 3 — 2026-09-22)

Per user descoping decision, the outcome of every item in the master list:

| # | Item | Disposition |
| --- | --- | --- |
| 1-3 | node template: `stop_enabled`, `edge_location_configs`, `clm_networking_mode` | **DESCOPED** by user decision — not implemented in this branch |
| 4 | rebalancing: `evict_gracefully` | **CLOSED** — `castai_rebalancing_schedule.launch_configuration.evict_gracefully` (create/update/read wired; unit test) |
| 5 | rebalancing: `ignore_problem_prevented_drain_pods` | **CLOSED** — `...aggressive_mode_config.ignore_problem_prevented_drain_pods` |
| 6 | V2 policies: `enabled`/`scoped_mode` zero-value bug | **CLOSED** — `toPoliciesV2` now reads raw-config presence, explicit `false` reaches the PUT body (tests added) |
| 7 | rebalancing: `max_simultaneous_drains` | **CLOSED** — `castai_rebalancing_schedule.launch_configuration.max_simultaneous_drains` |
| 8-9 | node template: `stuck_pod_resize_reconciliation`, `lifecycle_taints_disabled` | **DESCOPED** by user decision — not implemented |
| 10 | evictor: `cleanup_karpenter_nodes` | **NOT IMPLEMENTED BY DESIGN** — the field exists only in the deprecated V1 policies API; the standalone Evictor API (`castai/sdk/workload_eviction.Config`, used by `castai_evictor`) has no counterpart. Per user decision, Evictor is managed through `castai_evictor`, and the deprecated V1 surface is left unchanged. |
| 11 | evictor: `soft_tainting` | **ALREADY COVERED** — `castai_evictor` already exposes `soft_tainting` (the Phase 2 analysis compared the policies SDK surfaces only and missed the `workload_eviction` surface; `castai_evictor` is at full parity with `workload_eviction.Config` — all 24 config fields exposed) |
| 12 | spot diversity JSON key (`spotDiversityPriceIncrease` → `...LimitPercent`) | **CLOSED** — `castai/types/autoscaler_policies.go` mirror renamed; regression test added |
| 13 | deprecate `edge_location_ids` | **DESCOPED** (depends on item 2) |
| 14 | deprecate `keep_drain_timeout_nodes` | **CLOSED** — now carries `Deprecated: "Use evict_gracefully instead."` |
| 15-16 | `DiskGibToCpuRatio`, `Evictor.Allowed` | **NOT ADDED** (deprecated; recommendation unchanged) |

**Descope decision (user, 2026-09-22):** node-template gaps (items 1-3, 8-9, 13) are intentionally skipped; policies v1 (`resource_autoscaler.go`) is kept to a minimum (JSON-key bugfix only) because the resource is deprecated in favor of `castai_autoscaler_policies` + `castai_evictor`; Evictor configuration belongs to the separate `castai_evictor` resource.

**Provider docs** regenerated via `make generate-docs` reflecting the new rebalancing schema (`docs/resources/rebalancing_schedule.md`, `docs/data-sources/rebalancing_schedule.md`).

**Note:** this file lives in `templates/` because `tfplugindocs generate` (run by `make generate-docs`) deletes non-generated files under `docs/`; files in `templates/` are copied back on each regeneration.

### Master priority list (all GAP/MISMATCH items across all three areas)

**High priority**

1. **GAP** `sdk.NodetemplatesV1NewNodeTemplate.StopEnabled` → `castai_node_template.stop_enabled` (Storage Optimization).
2. **GAP** `sdk.NodetemplatesV1NewNodeTemplate.EdgeLocationConfigs` → `castai_node_template.edge_location_configs` (list of `{edge_location_id, edge_config_id}`); pairs with deprecating `edge_location_ids`.
3. **GAP** `sdk.NodetemplatesV1NewNodeTemplate.ClmNetworkingMode` → `castai_node_template.clm_networking_mode` (`default`/`none`/`tc`/`cni`).
4. **GAP** `sdk.ScheduledrebalancingV1RebalancingOptions.EvictGracefully` → `castai_rebalancing_schedule.launch_configuration.evict_gracefully`; pairs with deprecating `keep_drain_timeout_nodes`.
5. **GAP** `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemPreventedDrainPods` → `castai_rebalancing_schedule.launch_configuration.aggressive_mode_config.ignore_problem_prevented_drain_pods`.
6. **MISMATCH** `cluster_autoscaler_v2.PoliciesV2.Enabled` / `.ScopedMode`: `toPoliciesV2` uses `data.GetOk` so `enabled = false` / `scoped_mode = false` are silently dropped from the PUT body (resource_autoscaler_policies.go:262/267) — the global master switch cannot be disabled.

**Medium priority**

7. **GAP** `sdk.ScheduledrebalancingV1RebalancingOptions.MaxSimultaneousDrains` → `launch_configuration.max_simultaneous_drains`.
8. **GAP** `sdk.NodetemplatesV1NewNodeTemplate.StuckPodResizeReconciliation` → `castai_node_template.stuck_pod_resize_reconciliation` block (`enabled`).
9. **GAP** `sdk.NodetemplatesV1TemplateConstraints.LifecycleTaintsDisabled` → `castai_node_template.constraints.lifecycle_taints_disabled`.
10. **GAP** `sdk.PoliciesV1Evictor.CleanupKarpenterNodes` → `castai_autoscaler...evictor.cleanup_karpenter_nodes` (+ `types.Evictor` mirror).
11. **GAP** `sdk.PoliciesV1Evictor.SoftTainting` → `...evictor.soft_tainting` (+ mirror).

**Low priority / correctness fixes**

12. **MISMATCH** `types.AutoscalerPolicy` json key `spotDiversityPriceIncrease` → rename to `spotDiversityPriceIncreaseLimitPercent` (currently silently no-ops; castai/types/autoscaler_policies.go).
13. **MISMATCH** deprecate `castai_node_template.edge_location_ids` once item 2 lands.
14. **MISMATCH** deprecate `castai_rebalancing_schedule...keep_drain_timeout_nodes` once item 4 lands.
15. **GAP** `sdk.PoliciesV1UnschedulablePodsPolicy.DiskGibToCpuRatio`: deprecated + API-ignored — document as unsupported, do not add.
16. **GAP** `sdk.PoliciesV1Evictor.Allowed`: deprecated — do not add.

Detailed per-area tables follow.

## Scope

Field-by-field parity comparison between the regenerated SDK and the Terraform provider resource schemas for the autoscaler parity areas. This first part covers the SDK-root-based resources:

1. `castai_autoscaler` (`castai/resource_autoscaler.go`) — PoliciesAPI
2. `castai_node_template` (`castai/resource_node_template.go`) — NodeTemplatesAPI
3. `castai_rebalancing_schedule` (`castai/resource_rebalancing_schedule.go`) — ScheduledRebalancingAPI
4. `castai_rebalancing_job` (`castai/resource_rebalancing_job.go`) — ScheduledRebalancingAPI

Out of scope (handled by other steps/workers): `cluster_autoscaler_v2` / `resource_autoscaler_policies.go`, node configuration resources (`NodeconfigV1*`), and evictor-specific resources.

- Spec source: api.cast.ai, SDK regenerated 2026-09-22, commit `360ecff`.
- Analysis worktree: branch `sdk-regen-autoscaler-parity`.

### Tag legend

| Tag | Meaning |
| --- | --- |
| GAP | Field exists in the SDK but has no corresponding TF schema attribute |
| MISMATCH | Field exists in both but with different type/name/optionality, or the SDK deprecates the field the TF side still uses exclusively |
| OK | Field is mapped correctly (name only, no padding) |

Note on `castai_node_template_defaults`: **no such resource exists in the codebase.** `castai/provider.go` registers only `castai_node_template` (line 72). The file `castai/resource_node_template_defaults_test.go` exists, but it tests `resourceNodeTemplate()` (the `default-by-castai` default node template path) — it does not define or reference a `castai_node_template_defaults` resource. The analysis below therefore covers the single registered `castai_node_template` resource, which also serves as the "defaults" (default node template) surface.

Note on the new `ExternalclusterV1DrainConfig.Pods` field: grepped all four in-scope resource files for `DrainConfig` — **zero hits**. `ExternalclusterV1DrainConfig` (including the new `Pods *[]ClusteractionsV1ClusterActionDrainNodePodRef` partial-drain field) is only referenced by `ExternalClusterAPIDrainNode` (`ExternalClusterAPIDrainNodeJSONRequestBody`, api.gen.go:13829), which is not mapped by any of the five in-scope resources (nor, in fact, by any provider file — no provider code references `ExternalClusterAPIDrainNode` today). No action needed for this step.

---

## `resource_autoscaler` (castai_autoscaler)

Request body type: `PoliciesAPIUpsertClusterPoliciesJSONRequestBody = sdk.PoliciesV1Policies` (api.gen.go:13787). The resource does not build the SDK struct directly; it serializes a local mirror type (`castai/types.AutoscalerPolicy`, castai/types/autoscaler_policies.go) to JSON, merges it with the current policies via JSON merge-patch, and PUTs the raw JSON. Response is read as raw bytes (`PoliciesAPIGetClusterPolicies`). This indirection means the mirror type's JSON tags must be compared against the SDK's JSON tags — one mismatch was found (see below).

`castai_autoscaler` itself is deprecated (`DeprecationMessage` at resourceAutoscaler) in favor of `castai_autoscaler_policies` / `castai_evictor`, so gaps here are lower priority by design.

### `sdk.PoliciesV1Policies` (top level)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.PoliciesV1Policies.ClusterLimits` | `autoscaler_settings.cluster_limits` | OK | |
| `sdk.PoliciesV1Policies.DefaultNodeTemplateVersion` | — | OK | Computed; deliberately stripped by `filterVolatileFields` |
| `sdk.PoliciesV1Policies.Enabled` | `autoscaler_settings.enabled` | OK | |
| `sdk.PoliciesV1Policies.IsScopedMode` | `autoscaler_settings.is_scoped_mode` | OK | |
| `sdk.PoliciesV1Policies.NodeDownscaler` | `autoscaler_settings.node_downscaler` | OK | |
| `sdk.PoliciesV1Policies.NodeTemplatesPartialMatchingEnabled` | `autoscaler_settings.node_templates_partial_matching_enabled` | OK | |
| `sdk.PoliciesV1Policies.SpotInstances` | `autoscaler_settings.spot_instances` | OK | Deprecated on both sides |
| `sdk.PoliciesV1Policies.UnschedulablePods` | `autoscaler_settings.unschedulable_pods` | OK | |

### `sdk.PoliciesV1ClusterLimitsPolicy` / `PoliciesV1ClusterLimitsCpu`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.PoliciesV1ClusterLimitsPolicy.Cpu` | `cluster_limits.cpu` | OK | |
| `sdk.PoliciesV1ClusterLimitsPolicy.Enabled` | `cluster_limits.enabled` | OK | |
| `sdk.PoliciesV1ClusterLimitsCpu.MaxCores` | `cluster_limits.cpu.max_cores` | OK | |
| `sdk.PoliciesV1ClusterLimitsCpu.MinCores` | `cluster_limits.cpu.min_cores` | OK | Deprecated on both sides |

### `sdk.PoliciesV1UnschedulablePodsPolicy`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.PoliciesV1UnschedulablePodsPolicy.CustomInstancesEnabled` | `unschedulable_pods.custom_instances_enabled` | OK | Deprecated on both sides |
| `sdk.PoliciesV1UnschedulablePodsPolicy.DiskGibToCpuRatio` | — | **GAP** | Deprecated ("input only (for backwards-compatibility, ignored)") and absent from the `types.AutoscalerPolicy` mirror too. TF schema block: `castai/resource_autoscaler.go` inside `FieldUnschedulablePods` (~line 143). Low priority — SDK says the field is ignored by the API |
| `sdk.PoliciesV1UnschedulablePodsPolicy.Enabled` | `unschedulable_pods.enabled` | OK | |
| `sdk.PoliciesV1UnschedulablePodsPolicy.Headroom` | `unschedulable_pods.headroom` | OK | Deprecated on both sides |
| `sdk.PoliciesV1UnschedulablePodsPolicy.HeadroomSpot` | `unschedulable_pods.headroom_spot` | OK | Deprecated on both sides |
| `sdk.PoliciesV1UnschedulablePodsPolicy.NodeConstraints` | `unschedulable_pods.node_constraints` | OK | Deprecated on both sides |
| `sdk.PoliciesV1UnschedulablePodsPolicy.PodPinner` | `unschedulable_pods.pod_pinner` | OK | |

Nested (all OK): `PoliciesV1Headroom.CpuPercentage`/`Enabled`/`MemoryPercentage` → `cpu_percentage`/`enabled`/`memory_percentage`; `PoliciesV1NodeConstraints.Enabled`/`MaxCpuCores`/`MaxRamMib`/`MinCpuCores`/`MinRamMib` → `enabled`/`max_cpu_cores`/`max_ram_mib`/`min_cpu_cores`/`min_ram_mib`; `PoliciesV1PodPinner.Enabled` → `pod_pinner.enabled` (`PodPinner.Status` is computed and stripped by `filterVolatileFields`).

### `sdk.PoliciesV1SpotInstances`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.PoliciesV1SpotInstances.Enabled` | `spot_instances.enabled` | OK | Deprecated on both sides |
| `sdk.PoliciesV1SpotInstances.MaxReclaimRate` | `spot_instances.max_reclaim_rate` | OK | Deprecated on both sides |
| `sdk.PoliciesV1SpotInstances.SpotBackups` | `spot_instances.spot_backups` | OK | |
| `sdk.PoliciesV1SpotInstances.SpotDiversityEnabled` | `spot_instances.spot_diversity_enabled` | OK | Deprecated on both sides |
| `sdk.PoliciesV1SpotInstances.SpotDiversityPriceIncreaseLimitPercent` | `spot_instances.spot_diversity_price_increase_limit` | **MISMATCH** | JSON key renamed in the API: SDK field is `spotDiversityPriceIncreaseLimitPercent`, but the provider's mirror `types.AutoscalerPolicy.SpotInstances.SpotDiversityPriceIncrease` serializes json key `spotDiversityPriceIncrease` (castai/types/autoscaler_policies.go). The upsert body therefore carries a key the API no longer reads — the setting silently has no effect. Fix: rename the mirror's json tag to `spotDiversityPriceIncreaseLimitPercent`. TF schema block: `castai/resource_autoscaler.go:369` (`FieldSpotDiversityPriceIncreaseLimit`, itself deprecated in favor of node-template `constraints.spot_diversity_price_increase_limit_percent`) |
| `sdk.PoliciesV1SpotInstances.SpotInterruptionPredictions` | `spot_instances.spot_interruption_predictions` | OK | Deprecated on both sides |

Nested (all OK): `PoliciesV1SpotBackups.Enabled`/`SpotBackupRestoreRateSeconds` → `enabled`/`spot_backup_restore_rate_seconds`; `PoliciesV1SpotInterruptionPredictions.Enabled`/`Type` → `enabled`/`spot_interruption_predictions_type`.

### `sdk.PoliciesV1NodeDownscaler` / `PoliciesV1Evictor`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.PoliciesV1NodeDownscaler.Enabled` | `node_downscaler.enabled` | OK | |
| `sdk.PoliciesV1NodeDownscaler.EmptyNodes` | `node_downscaler.empty_nodes` | OK | |
| `sdk.PoliciesV1NodeDownscaler.Evictor` | `node_downscaler.evictor` | OK | |
| `sdk.PoliciesV1NodeDownscalerEmptyNodes.Enabled` | `node_downscaler.empty_nodes.enabled` | OK | |
| `sdk.PoliciesV1NodeDownscalerEmptyNodes.DelaySeconds` | `node_downscaler.empty_nodes.delay_seconds` | OK | |
| `sdk.PoliciesV1Evictor.AggressiveMode` | `node_downscaler.evictor.aggressive_mode` | OK | |
| `sdk.PoliciesV1Evictor.Allowed` | — | **GAP** | Deprecated ("use status instead") and absent from `types.Evictor` mirror. TF schema block: `castai/resource_autoscaler.go:443` (`FieldEvictor`). Low priority — deprecated legacy response flag |
| `sdk.PoliciesV1Evictor.CleanupKarpenterNodes` | — | **GAP** | New field: "If enabled, Evictor will delete Karpenter NodeClaims after draining Karpenter-managed nodes, triggering Karpenter's termination controller for fast instance cleanup." Absent from both the TF schema (`castai/resource_autoscaler.go:443`, `FieldEvictor` block) and the `types.Evictor` mirror (castai/types/autoscaler_policies.go). Medium priority (resource deprecated, but functional) |
| `sdk.PoliciesV1Evictor.CycleInterval` | `node_downscaler.evictor.cycle_interval` | OK | |
| `sdk.PoliciesV1Evictor.DryRun` | `node_downscaler.evictor.dry_run` | OK | |
| `sdk.PoliciesV1Evictor.Enabled` | `node_downscaler.evictor.enabled` | OK | |
| `sdk.PoliciesV1Evictor.IgnorePodDisruptionBudgets` | `node_downscaler.evictor.ignore_pod_disruption_budgets` | OK | |
| `sdk.PoliciesV1Evictor.NodeGracePeriodMinutes` | `node_downscaler.evictor.node_grace_period_minutes` | OK | |
| `sdk.PoliciesV1Evictor.PodEvictionFailureBackOffInterval` | `node_downscaler.evictor.pod_eviction_failure_back_off_interval` | OK | |
| `sdk.PoliciesV1Evictor.ScopedMode` | `node_downscaler.evictor.scoped_mode` | OK | |
| `sdk.PoliciesV1Evictor.SoftTainting` | — | **GAP** | New field: "If enabled, Evictor will use soft tainting (PreferNoSchedule) instead of hard cordoning after eviction." Absent from both the TF schema (`castai/resource_autoscaler.go:443`, `FieldEvictor` block) and the `types.Evictor` mirror (castai/types/autoscaler_policies.go). Medium priority |
| `sdk.PoliciesV1Evictor.Status` | — | OK | Computed; stripped by `filterVolatileFields` |

Autoscaler totals: ~47 request-body fields compared across 10 SDK structs. 4 GAPs, 1 MISMATCH.

---

## `resource_node_template` (castai_node_template; covers the default/defaults surface)

Request body types: `NodeTemplatesAPICreateNodeTemplateJSONRequestBody = sdk.NodetemplatesV1NewNodeTemplate` (api.gen.go:13781) and `NodeTemplatesAPIUpdateNodeTemplateJSONRequestBody = sdk.NodetemplatesV1UpdateNodeTemplate` (api.gen.go:13784). Response type: `sdk.NodetemplatesV1NodeTemplate` (read via `NodeTemplatesAPIListNodeTemplates`). The `castai_node_template_defaults` test file exercises the same resource with `name = "default-by-castai"` — see scope note above.

### Top level (`sdk.NodetemplatesV1NewNodeTemplate` / `NodetemplatesV1UpdateNodeTemplate`)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.NodetemplatesV1NewNodeTemplate.ClmEnabled` | `clm_enabled` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.ClmNetworkingMode` | — | **GAP** | CLM networking mode: `"default"` (auto-detect), `"none"` (no connection preservation), `"tc"` (new pod IP on destination, eBPF traffic-control translation of old IP; cross-subnet/AZ migration), `"cni"` (pod IP preservation via CNI; requires same subnet/AZ). Missing from both create and update paths. TF schema block: `castai/resource_node_template.go:879` (next to `FieldNodeTemplateClmEnabled`). **High priority** — the only way to configure an already-shipped CLM feature |
| `sdk.NodetemplatesV1NewNodeTemplate.ConfigurationId` | `configuration_id` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.Constraints` | `constraints` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.CustomInstancesEnabled` | `custom_instances_enabled` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.CustomInstancesWithExtendedMemoryEnabled` | `custom_instances_with_extended_memory_enabled` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.CustomLabel` | — | OK | Legacy single label; superseded by `customLabels`. Provider intentionally uses `CustomLabels` only |
| `sdk.NodetemplatesV1NewNodeTemplate.CustomLabels` | `custom_labels` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.CustomTaints` | `custom_taints` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.EdgeLocationConfigs` | — | **GAP** | `[]NodetemplatesV1EdgeLocationConfig` pairing an edge location with an optional `edgeConfigId` (`NodetemplatesV1EdgeLocationConfig{EdgeConfigId, EdgeLocationId}`, api.gen.go:7784). The SDK deprecates `EdgeLocationIds` in its favor. TF schema block: `castai/resource_node_template.go:885` (currently only `FieldNodeTemplateEdgeLocationIDs`). **High priority** — needed to attach edge configurations (see commit 5524725 "edge configuration added for Nebius") |
| `sdk.NodetemplatesV1NewNodeTemplate.EdgeLocationIds` | `edge_location_ids` | **MISMATCH** | Mapped, but the SDK marks it "Deprecated: use edge_location_configs instead". TF exposes only the deprecated field; the replacement (`EdgeLocationConfigs`) is unexposed, so users cannot configure the edge config id at all |
| `sdk.NodetemplatesV1NewNodeTemplate.Gpu` | `gpu` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.IsDefault` | `is_default` | OK | TF: Optional+Computed, validated against name (`default-by-castai`) |
| `sdk.NodetemplatesV1NewNodeTemplate.IsEnabled` | `is_enabled` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.Name` | `name` | OK | Create-only (`ForceNew` in TF); absent from `NodetemplatesV1UpdateNodeTemplate` — consistent |
| `sdk.NodetemplatesV1NewNodeTemplate.PriceAdjustmentConfiguration` | `price_adjustment_configuration` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.RebalancingConfig` | `rebalancing_config_min_nodes` | OK | Flattened to a single int (`MinNodes`) |
| `sdk.NodetemplatesV1NewNodeTemplate.ShouldTaint` | `should_taint` | OK | |
| `sdk.NodetemplatesV1NewNodeTemplate.StopEnabled` | — | **GAP** | "Nodes in this node-template have Storage Optimization (STOP) enabled." Missing from both create and update paths. TF schema block: `castai/resource_node_template.go:879` (top-level schema, next to `clm_enabled`). **High priority** |
| `sdk.NodetemplatesV1NewNodeTemplate.StuckPodResizeReconciliation` | — | **GAP** | `NodetemplatesV1StuckPodResizeReconciliation{Enabled *bool}`: "When enabled, the autoscaler discovers pods whose woop-initiated in-place resize failed or got stuck, protects them from woop's eviction fallback, and live-migrates them via a rebalancing plan." TF schema block: `castai/resource_node_template.go:879` (top-level schema). Medium priority |

Response-only (computed, summarized): `NodetemplatesV1NodeTemplate.ConfigurationName`, `NodetemplatesV1NodeTemplate.Version` — not exposed in TF state (fine; `Version` is used implicitly for optimistic concurrency errors only).

### `sdk.NodetemplatesV1TemplateConstraints` (constraints block, `castai/resource_node_template.go:252`)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.NodetemplatesV1TemplateConstraints.ArchitecturePriority` | `constraints.architecture_priority` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Architectures` | `constraints.architectures` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Aws` | `constraints.aws` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Azs` | `constraints.azs` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.BareMetal` | `constraints.bare_metal` | OK | TF string tri-state `unspecified/true/false` ↔ SDK `*bool` |
| `sdk.NodetemplatesV1TemplateConstraints.Burstable` | `constraints.burstable_instances` | OK | TF string tri-state ↔ `ConstraintState` enum |
| `sdk.NodetemplatesV1TemplateConstraints.ComputeOptimized` | `constraints.compute_optimized_state` | OK | Legacy `compute_optimized` bool kept for migration |
| `sdk.NodetemplatesV1TemplateConstraints.CpuManufacturers` | `constraints.cpu_manufacturers` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.CustomPriority` | `constraints.custom_priority` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.CustomerSpecific` | `constraints.customer_specific` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.DedicatedNodeAffinity` | `constraints.dedicated_node_affinity` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.EnableSpotDiversity` | `constraints.enable_spot_diversity` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.EnableSpotReliability` | `constraints.spot_reliability_enabled` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.FallbackRestoreRateSeconds` | `constraints.fallback_restore_rate_seconds` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Gcp` | `constraints.gcp` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Gpu` | `constraints.gpu` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.InstanceFamilies` | `constraints.instance_families` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.IsGpuOnly` | `constraints.is_gpu_only` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.LifecycleTaintsDisabled` | — | **GAP** | "Whether lifecycle taints should be applied." No TF attribute. TF schema block: `castai/resource_node_template.go:252` (`constraints` elem schema, e.g. next to `FieldNodeTemplateBareMetal` at line 514). Medium priority |
| `sdk.NodetemplatesV1TemplateConstraints.MaxCpu` | `constraints.max_cpu` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.MaxMemory` | `constraints.max_memory` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.MaxPricePerCpu` | `constraints.max_price_per_cpu` | OK | `float64` ↔ TypeFloat |
| `sdk.NodetemplatesV1TemplateConstraints.MinCpu` | `constraints.min_cpu` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.MinMemory` | `constraints.min_memory` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.OnDemand` | `constraints.on_demand` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Os` | `constraints.os` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.ResourceLimits` | `constraints.resource_limits` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.Spot` | `constraints.spot` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.SpotDiversityPriceIncreaseLimitPercent` | `constraints.spot_diversity_price_increase_limit_percent` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.SpotInterruptionPredictionsEnabled` | `constraints.spot_interruption_predictions_enabled` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.SpotInterruptionPredictionsType` | `constraints.spot_interruption_predictions_type` | OK | Diff-suppressed; TF description documents deprecation of `aws-rebalance-recommendations` |
| `sdk.NodetemplatesV1TemplateConstraints.SpotReliabilityPriceIncreaseLimitPercent` | `constraints.spot_reliability_price_increase_limit_percent` | OK | |
| `sdk.NodetemplatesV1TemplateConstraints.StorageOptimized` | `constraints.storage_optimized_state` | OK | Legacy `storage_optimized` bool kept for migration |

Nested constraint structs (all fully covered, OK):
- `AWSConstraints.CapacityReservations[].{Id, CapacityResourceGroupArn, Type}` → `aws.capacity_reservations[].{id, capacity_resource_group_arn, type}`
- `GCPConstraints.CapacityReservationIds` → `gcp.capacity_reservation_ids`
- `GPUConstraints.{ExcludeNames, FractionalGpus, IncludeNames, Manufacturers, MaxCount, MinCount}` → `gpu.{exclude_names, fractional_gpus, include_names, manufacturers, max_count, min_count}`
- `InstanceFamilyConstraints.{Exclude, Include}` → `instance_families.{exclude, include}`
- `CustomPriority[].{Families, OnDemand, Spot}` → `custom_priority[].{instance_families, on_demand, spot}`
- `DedicatedNodeAffinity[].{Affinity, AzName, CpusPerGpu, InstanceTypes, MaxCpu, MinGpusPerNode, Name}` → `dedicated_node_affinity[].{affinity, az_name, cpus_per_gpu, instance_types, max_cpu, min_gpus_per_node, name}` (affinity: `K8sSelectorV1KubernetesNodeAffinity.{Key, Operator, Values}` → `affinity[].{key, operator, values}`)
- `ResourceLimits.{CpuLimitEnabled, CpuLimitMaxCores}` → `resource_limits.{cpu_limit_enabled, cpu_limit_max_cores}`

### Other node-template blocks (all OK)

- `sdk.NodetemplatesV1GPU.{DefaultSharedClientsPerGpu, EnableTimeSharing, SharingConfiguration, SharingStrategy, UserManagedGpuDrivers}` → `gpu.{default_shared_clients_per_gpu, enable_time_sharing, sharing_configuration, sharing_strategy, user_managed_gpu_drivers}` (enable_time_sharing deprecated on both sides)
- `sdk.NodetemplatesV1TaintWithOptionalEffect.{Key, Value, Effect}` → `custom_taints[].{key, value, effect}`
- `sdk.NodetemplatesV1RebalancingConfiguration.MinNodes` → `rebalancing_config_min_nodes`
- `sdk.NodetemplatesV1PriceAdjustmentConfiguration.InstanceTypeAdjustments` → `price_adjustment_configuration.instance_type_adjustments`

Node template totals: ~84 request-body fields compared across 20+ SDK structs. 5 GAPs, 1 MISMATCH.

---

## `rebalancing` (castai_rebalancing_schedule + castai_rebalancing_job)

Request body types: `ScheduledRebalancingAPICreateRebalancingScheduleJSONRequestBody = sdk.ScheduledrebalancingV1RebalancingSchedule` and `ScheduledRebalancingAPIUpdateRebalancingScheduleJSONRequestBody = sdk.ScheduledrebalancingV1RebalancingScheduleUpdate` for the schedule; `ScheduledRebalancingAPICreateRebalancingJobJSONRequestBody` / `ScheduledRebalancingAPIUpdateRebalancingJobJSONRequestBody = sdk.ScheduledrebalancingV1RebalancingJob` for the job. Response types: `ScheduledrebalancingV1RebalancingSchedule`, `ScheduledrebalancingV1RebalancingJob`.

### castai_rebalancing_schedule — top level

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.Name` | `name` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.Schedule` | `schedule` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.LaunchConfiguration` | `launch_configuration` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.TriggerConditions` | `trigger_conditions` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.Id` | — | OK | Resource ID |
| `sdk.ScheduledrebalancingV1RebalancingSchedule.Jobs` / `LastTriggerAt` / `NextTriggerAt` | — | OK | Response-only, computed |

### `sdk.ScheduledrebalancingV1Schedule` / `TriggerConditions` / `LaunchConfiguration`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1Schedule.Cron` | `schedule.cron` | OK | CRON_TZ support documented in TF |
| `sdk.ScheduledrebalancingV1TriggerConditions.SavingsPercentage` | `trigger_conditions.savings_percentage` | OK | `float32` ↔ TypeFloat (5-digit truncation handled) |
| `sdk.ScheduledrebalancingV1TriggerConditions.IgnoreSavings` | `trigger_conditions.ignore_savings` | OK | |
| `sdk.ScheduledrebalancingV1LaunchConfiguration.NodeTtlSeconds` | `launch_configuration.node_ttl_seconds` | OK | |
| `sdk.ScheduledrebalancingV1LaunchConfiguration.NumTargetedNodes` | `launch_configuration.num_targeted_nodes` | OK | |
| `sdk.ScheduledrebalancingV1LaunchConfiguration.RebalancingOptions` | `launch_configuration` (flattened) | OK | See below |
| `sdk.ScheduledrebalancingV1LaunchConfiguration.Selector` | `launch_configuration.selector` | OK | JSON-encoded node selector (`NodeSelectorTerms[].MatchExpressions/MatchFields[].{Key, Operator, Values}`) |
| `sdk.ScheduledrebalancingV1LaunchConfiguration.TargetNodeSelectionAlgorithm` | `launch_configuration.target_node_selection_algorithm` | OK | |

### `sdk.ScheduledrebalancingV1RebalancingOptions` (flattened into `launch_configuration`)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1RebalancingOptions.AggressiveMode` | `launch_configuration.aggressive_mode` | OK | Deprecated on both sides |
| `sdk.ScheduledrebalancingV1RebalancingOptions.AggressiveModeConfig` | `launch_configuration.aggressive_mode_config` | OK | See field table below — one nested GAP |
| `sdk.ScheduledrebalancingV1RebalancingOptions.DrainFailureConfig` | `launch_configuration.drain_failure_config` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingOptions.EvictGracefully` | — | **GAP** | "Defines whether the nodes that failed to get drained until a predefined timeout, will be kept with a rebalancing.cast.ai/status=drain-failed annotation instead of forcefully drained." This is the SDK's **replacement** for the deprecated `KeepDrainTimeoutNodes` (see MISMATCH below). TF schema block: `castai/resource_rebalancing_schedule.go:92` (`launch_configuration` elem schema, next to `keep_drain_timeout_nodes` at line 116). **High priority** |
| `sdk.ScheduledrebalancingV1RebalancingOptions.ExecutionConditions` | `launch_configuration.execution_conditions` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingOptions.KeepDrainTimeoutNodes` | `launch_configuration.keep_drain_timeout_nodes` | **MISMATCH** | Mapped, but SDK marks it "Deprecated: use evictGracefully instead" — the replacement field is unexposed, so the provider can only drive the deprecated path |
| `sdk.ScheduledrebalancingV1RebalancingOptions.MaxSimultaneousDrains` | — | **GAP** | "Number of nodes to drain simultaneously. When unspecified, defaults to unlimited." TF schema block: `castai/resource_rebalancing_schedule.go:92` (`launch_configuration` elem schema). Medium priority |
| `sdk.ScheduledrebalancingV1RebalancingOptions.MinNodes` | `launch_configuration.rebalancing_min_nodes` | OK | |

### `sdk.ScheduledrebalancingV1AggressiveModeConfig`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreLocalPersistentVolumes` | `aggressive_mode_config.ignore_local_persistent_volumes` | OK | |
| `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemJobPods` | `aggressive_mode_config.ignore_problem_job_pods` | OK | |
| `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemPodsWithoutController` | `aggressive_mode_config.ignore_problem_pods_without_controller` | OK | |
| `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemPreventedDrainPods` | — | **GAP** | "Pods annotated with rebalancing.cast.ai/prevented-drain=true will not prevent the Rebalancer from deleting a node on which they run." Missing from `stateToSchedule` (castai/resource_rebalancing_schedule.go), `scheduleToState`, and the schema. TF schema block: `castai/resource_rebalancing_schedule.go:127` (`aggressive_mode_config` elem schema). **High priority** — a live request-body knob |
| `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemRemovalDisabledPods` | `aggressive_mode_config.ignore_problem_removal_disabled_pods` | OK | |

### `sdk.ScheduledrebalancingV1ExecutionConditions` / `DrainFailureConfig`

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1ExecutionConditions.Enabled` | `execution_conditions.enabled` | OK | |
| `sdk.ScheduledrebalancingV1ExecutionConditions.AchievedSavingsPercentage` | `execution_conditions.achieved_savings_percentage` | OK | |
| `sdk.ScheduledrebalancingV1DrainFailureConfig.DisableUncordon` | `drain_failure_config.disable_uncordon` | OK | |
| `sdk.ScheduledrebalancingV1DrainFailureConfig.UncordonAfterSeconds` | `drain_failure_config.uncordon_after_seconds` | OK | TF enforces the SDK's clamp range (60–259200) |

### castai_rebalancing_job

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `sdk.ScheduledrebalancingV1RebalancingJob.ClusterId` | `cluster_id` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingJob.Enabled` | `enabled` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingJob.RebalancingScheduleId` | `rebalancing_schedule_id` | OK | |
| `sdk.ScheduledrebalancingV1RebalancingJob.Id` | — | OK | Resource ID |
| `sdk.ScheduledrebalancingV1RebalancingJob.LastTriggerAt` / `NextTriggerAt` / `RebalancingPlanId` / `Status` | — | OK (response-only, summarized) | Computed execution-tracking fields; not exposed in TF state. Optional future enhancement: expose as Computed attributes |

Rebalancing totals: ~41 request-body fields compared (schedule ~34, job ~7). 3 GAPs, 1 MISMATCH.

---

## Summary of gaps to implement (SDK-root resources — detail; see consolidated master list at top)

Priority-ordered list of all GAP/MISMATCH items (the only actionable findings):

### High priority

1. **GAP** — `sdk.NodetemplatesV1NewNodeTemplate.StopEnabled` → add `stop_enabled` to `castai_node_template` (castai/resource_node_template.go:879, top-level schema; wire into create + update + read).
2. **GAP** — `sdk.NodetemplatesV1NewNodeTemplate.EdgeLocationConfigs` (+ fix MISMATCH item 8 below in the same change) → add `edge_location_configs` (list of `{edge_location_id, edge_config_id}`) to `castai_node_template` (castai/resource_node_template.go:885).
3. **GAP** — `sdk.NodetemplatesV1NewNodeTemplate.ClmNetworkingMode` → add `clm_networking_mode` (string, values `default`/`none`/`tc`/`cni`) to `castai_node_template` (castai/resource_node_template.go:879).
4. **GAP** — `sdk.ScheduledrebalancingV1RebalancingOptions.EvictGracefully` → add `evict_gracefully` to `castai_rebalancing_schedule.launch_configuration` (castai/resource_rebalancing_schedule.go:92); pairs with deprecating `keep_drain_timeout_nodes` (MISMATCH item 9).
5. **GAP** — `sdk.ScheduledrebalancingV1AggressiveModeConfig.IgnoreProblemPreventedDrainPods` → add `ignore_problem_prevented_drain_pods` to `castai_rebalancing_schedule.launch_configuration.aggressive_mode_config` (castai/resource_rebalancing_schedule.go:127).

### Medium priority

6. **GAP** — `sdk.ScheduledrebalancingV1RebalancingOptions.MaxSimultaneousDrains` → add `max_simultaneous_drains` to `castai_rebalancing_schedule.launch_configuration` (castai/resource_rebalancing_schedule.go:92).
7. **GAP** — `sdk.NodetemplatesV1NewNodeTemplate.StuckPodResizeReconciliation` → add `stuck_pod_resize_reconciliation` block (`enabled` bool) to `castai_node_template` (castai/resource_node_template.go:879).
8. **GAP** — `sdk.NodetemplatesV1TemplateConstraints.LifecycleTaintsDisabled` → add `lifecycle_taints_disabled` to `castai_node_template.constraints` (castai/resource_node_template.go:252, elem schema near line 514).
9. **GAP** — `sdk.PoliciesV1Evictor.CleanupKarpenterNodes` → add `cleanup_karpenter_nodes` to `castai_autoscaler.autoscaler_settings.node_downscaler.evictor` (castai/resource_autoscaler.go:443) and to the `types.Evictor` mirror.
10. **GAP** — `sdk.PoliciesV1Evictor.SoftTainting` → add `soft_tainting` to the same evictor block (castai/resource_autoscaler.go:443) and mirror.

### Low priority / correctness fixes

11. **MISMATCH** — `sdk.PoliciesV1SpotInstances.SpotDiversityPriceIncreaseLimitPercent` vs `types.AutoscalerPolicy` json key `spotDiversityPriceIncrease`: rename the mirror's json tag to `spotDiversityPriceIncreaseLimitPercent` so the upsert body key matches the API (castai/types/autoscaler_policies.go). Field is deprecated in TF, but currently silently no-ops.
12. **MISMATCH** — `sdk.NodetemplatesV1NewNodeTemplate.EdgeLocationIds` is deprecated in favor of `EdgeLocationConfigs`; once item 2 lands, deprecate `edge_location_ids` in TF.
13. **MISMATCH** — `sdk.ScheduledrebalancingV1RebalancingOptions.KeepDrainTimeoutNodes` is deprecated in favor of `EvictGracefully`; once item 4 lands, deprecate `keep_drain_timeout_nodes` in TF.
14. **GAP** — `sdk.PoliciesV1UnschedulablePodsPolicy.DiskGibToCpuRatio`: deprecated "input only, ignored" — recommend documenting as unsupported rather than adding.
15. **GAP** — `sdk.PoliciesV1Evictor.Allowed`: deprecated in favor of `status` — recommend not adding.

### Totals

| Resource | Fields compared (request-body) | GAP | MISMATCH |
| --- | --- | --- | --- |
| castai_autoscaler | ~47 | 4 | 1 |
| castai_node_template | ~84 | 5 | 1 |
| castai_rebalancing_schedule | ~34 | 3 | 1 |
| castai_rebalancing_job | ~7 | 0 | 0 |
| **Total** | **~172** | **12** | **3** |

No changes required for the new `ExternalclusterV1DrainConfig.Pods` field — none of the in-scope resources map DrainConfig (see scope note).

---

## cluster_autoscaler_v2 / resource_autoscaler_policies (castai_autoscaler_policies)

Field-by-field parity comparison between the regenerated `castai/sdk/cluster_autoscaler_v2` package (package `cluster_autoscaler_v2`, models.gen.go) and the `castai_autoscaler_policies` resource (`castai/resource_autoscaler_policies.go`).

Request body type: `PoliciesV2APIUpdateClusterPoliciesJSONRequestBody = cluster_autoscaler_v2.PoliciesV2` (models.gen.go:1537). Response type: `PoliciesV2APIGetClusterPoliciesResponse` wraps `*PoliciesV2`. Unlike the deprecated `castai_autoscaler` (V1, which serializes a local mirror type to raw JSON), the V2 resource maps TF state directly to and from the SDK structs (`toPoliciesV2` / `flatten*` helpers) — there is no mirror type to drift.

### Scope notes

1. **No `castai/policies` policy model exists.** `castai/policies/policy.go` (and `castai/policies/gke/policy.go`) contain AWS/GCP IAM-policy JSON templates (`GetIAMPolicy`, `GetUserInlinePolicy`, `GetManagedPolicies` via `go:embed`), used only by `castai/data_source_eks_settings.go` and `castai/data_source_gke.go`. They do **not** mirror the API `PoliciesV2` shape and have no autoscaler-policy parity surface. The task brief assumed otherwise; the actual PoliciesV2 mapping lives entirely in `castai/resource_autoscaler_policies.go`.
2. **V2 deliberately removed the deep V1 nesting.** The regenerated `PoliciesV2` (models.gen.go:1009) carries only six fields: `ClusterLimits`, `Enabled`, `NodeDownscaler`, `ScopedMode`, `UnschedulablePods`, `Version`. The V1-era areas named in the task brief — scheduled spot, headroom, spot instances, autodownscaler, workload autocreation, evictor, node constraints, custom instances, disk-to-CPU ratio — do **not** exist** in `cluster_autoscaler_v2.PoliciesV2`. Per the SDK doc comments (models.gen.go:1001-1008, 874-879, 1300-1310): headroom/spot_instances were removed as deprecated, node constraints and custom instances moved to the Node Template API, and the Evictor moved to the standalone Evictor API (covered by `castai_evictor`). These are intentional V2 API removals, not TF-side gaps.
3. **Regen drift:** the regenerated v2 `models.gen.go` changed only enum constant names (e.g. `RebalancingNodeResourceOfferingSPOT` is now `SPOT`); the policies structs' fields and JSON tags are unchanged. All findings below are therefore pre-existing TF-side issues, not regen drift.

### `cluster_autoscaler_v2.PoliciesV2` (top level, models.gen.go:1009)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `cluster_autoscaler_v2.PoliciesV2.ClusterLimits` | `cluster_limits` | OK | |
| `cluster_autoscaler_v2.PoliciesV2.Enabled` | `enabled` | **MISMATCH** | SDK field is `*bool` where explicit `false` is meaningful ("Enable/disable all policies"). The build path guards the top-level bools with `data.GetOk` (castai/resource_autoscaler_policies.go:262), which cannot distinguish unset from `false` (zero value) — a user configuration of `enabled = false` leaves `policies.Enabled = nil`, so the PUT body omits the flag, the API keeps the current value, and the setting silently never applies (recurring no-op in-place update on every apply). Same pattern at `scoped_mode` below. Fix in `toPoliciesV2` (castai/resource_autoscaler_policies.go:259, lines 262-268): make the bools Required (they have no documented default), or read presence from raw config instead of `GetOk`. Note the nested bools do not share this bug — `toClusterLimitsPolicy`/`toNodeDownscalerPolicy`/`toUnschedulablePodsPolicy` use map-key presence checks, which handle `false` correctly |
| `cluster_autoscaler_v2.PoliciesV2.NodeDownscaler` | `node_downscaler` | OK | |
| `cluster_autoscaler_v2.PoliciesV2.ScopedMode` | `scoped_mode` | **MISMATCH** | Same `data.GetOk` zero-value bug as `Enabled` (castai/resource_autoscaler_policies.go:267): `scoped_mode = false` is silently dropped from the request body. Same fix location |
| `cluster_autoscaler_v2.PoliciesV2.UnschedulablePods` | `unschedulable_pods` | OK | |
| `cluster_autoscaler_v2.PoliciesV2.Version` | `version` | OK | Computed in TF (castai/resource_autoscaler_policies.go:160); on update the last-read version is re-sent via `toPoliciesV2` for optimistic locking — correct round-trip |

### `cluster_autoscaler_v2.ClusterLimitsPolicy` / `ClusterLimitsCpu` (models.gen.go:458 / 448)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `cluster_autoscaler_v2.ClusterLimitsPolicy.Cpu` | `cluster_limits.cpu` | OK | |
| `cluster_autoscaler_v2.ClusterLimitsPolicy.Enabled` | `cluster_limits.enabled` | OK | Map-presence check handles explicit `false` correctly |
| `cluster_autoscaler_v2.ClusterLimitsCpu.MaxCores` | `cluster_limits.cpu.max_cores` | OK | Both required (`int32` json `maxCores` ↔ TypeInt Required); TF additionally validates `IntAtLeast(2)` |
| `cluster_autoscaler_v2.ClusterLimitsCpu.MinCores` | `cluster_limits.cpu.min_cores` | OK | Deprecated on both sides ("Min CPU limit is no longer enforced") |

### `cluster_autoscaler_v2.NodeDownscalerPolicy` (models.gen.go:874)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `cluster_autoscaler_v2.NodeDownscalerPolicy.EmptyNodesDelay` | `node_downscaler.empty_nodes_delay` | OK | Duration string (`*string` json `emptyNodesDelay` ↔ TypeString); no format validation in TF, but type-consistent |
| `cluster_autoscaler_v2.NodeDownscalerPolicy.EmptyNodesEnabled` | `node_downscaler.empty_nodes_enabled` | OK | Map-presence check handles `false` correctly |

### `cluster_autoscaler_v2.UnschedulablePodsPolicy` (models.gen.go:1300)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `cluster_autoscaler_v2.UnschedulablePodsPolicy.Enabled` | `unschedulable_pods.enabled` | OK | |
| `cluster_autoscaler_v2.UnschedulablePodsPolicy.PartialTemplateMatchingEnabled` | `unschedulable_pods.partial_template_matching_enabled` | OK | New in V2 (moved from top-level in V1); correctly exposed |
| `cluster_autoscaler_v2.UnschedulablePodsPolicy.PodPinner` | `unschedulable_pods.pod_pinner` | OK | |

### `cluster_autoscaler_v2.PodPinner` (models.gen.go:993)

| SDK field path | TF attribute | Tag | Notes |
| --- | --- | --- | --- |
| `cluster_autoscaler_v2.PodPinner.Enabled` | `unschedulable_pods.pod_pinner.enabled` | OK | |
| `cluster_autoscaler_v2.PodPinner.Status` | — | OK (response-only) | `*PodPinnerStatus` enum (`POD_PINNER_STATUS_COMPATIBLE` etc.) — computed by the API, not exposed in TF state (V1 strips it via `filterVolatileFields` too). Optional enhancement, not a gap |

### Summary of gaps to implement (policies v2)

The V2 request-body surface is fully covered — **zero GAPs**. The only actionable items are two zero-value MISMATCHes in the same function:

1. **MISMATCH (High priority)** — `cluster_autoscaler_v2.PoliciesV2.Enabled`: `enabled = false` is silently dropped because `toPoliciesV2` uses `data.GetOk` (castai/resource_autoscaler_policies.go:262). Users cannot disable the global master switch. Fix: read the value unconditionally/required, or use raw-config presence detection.
2. **MISMATCH (High priority)** — `cluster_autoscaler_v2.PoliciesV2.ScopedMode`: identical `data.GetOk` bug for `scoped_mode = false` (castai/resource_autoscaler_policies.go:267). Same fix, same change.

### Totals (policies v2)

| Resource | Fields compared | GAP | MISMATCH |
| --- | --- | --- | --- |
| castai_autoscaler_policies | 17 (across 6 SDK structs) | 0 | 2 |

Structs compared: `PoliciesV2` (6), `ClusterLimitsPolicy` (2), `ClusterLimitsCpu` (2), `NodeDownscalerPolicy` (2), `UnschedulablePodsPolicy` (3), `PodPinner` (2). This exhausts every struct reachable from `PoliciesV2` that carries user-settable policy config; the remaining `cluster_autoscaler_v2` types (Analysis, Rebalancing*, Hibernation*, NodeConfig, etc.) belong to other APIs (cluster advisor, rebalancing config, hibernation, node templates) and are not part of the policies surface.
