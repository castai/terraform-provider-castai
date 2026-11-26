# AKS permission configuration variants

`aks_permission_configurations.json` is a curated list of **valid ways to configure the Azure
permissions** required by Cast AI for the credentials (service principal or managed identity)
that connect an AKS cluster to Cast AI. Each entry is a self-contained permission-assignment
configuration intended to be fed to a test of Cast AI's credential permission validation:
the server must accept **every** entry.

These are only permission *assignments* — the JSON intentionally says nothing about how the
credentials are created or sent to Cast AI. Whether the principal authenticates with
`client_secret` (app registration) or `workload_identity` (federated identity) makes no
difference to permission validation; what matters is the set of role assignments on the
principal that `client_id` refers to.

## Baseline (officially advertised configuration)

The [castai/aks Terraform module](https://github.com/castai/terraform-castai-aks) (`iam.tf`) and
[docs.cast.ai cloud permissions](https://docs.cast.ai/docs/cloud-permissions) advertise a single
custom role (`CastAKSRole-<cluster>`) with these 36 actions, assigned to the cluster, node
(`MC_*`) and — when it differs — the VNet resource group:

```text
Microsoft.Compute/*/read
Microsoft.Compute/virtualMachines/*
Microsoft.Compute/virtualMachineScaleSets/*
Microsoft.Compute/disks/write
Microsoft.Compute/disks/delete
Microsoft.Compute/disks/beginGetAccess/action
Microsoft.Compute/galleries/write
Microsoft.Compute/galleries/delete
Microsoft.Compute/galleries/images/write
Microsoft.Compute/galleries/images/delete
Microsoft.Compute/galleries/images/versions/write
Microsoft.Compute/galleries/images/versions/delete
Microsoft.Compute/snapshots/write
Microsoft.Compute/snapshots/delete
Microsoft.Network/*/read
Microsoft.Network/networkInterfaces/write
Microsoft.Network/networkInterfaces/delete
Microsoft.Network/networkInterfaces/join/action
Microsoft.Network/networkSecurityGroups/join/action
Microsoft.Network/virtualNetworks/subnets/join/action
Microsoft.Network/applicationGateways/backendhealth/action
Microsoft.Network/applicationGateways/backendAddressPools/join/action
Microsoft.Network/applicationSecurityGroups/joinIpConfiguration/action
Microsoft.Network/loadBalancers/backendAddressPools/write
Microsoft.Network/loadBalancers/backendAddressPools/join/action
Microsoft.ContainerService/*/read
Microsoft.ContainerService/managedClusters/start/action
Microsoft.ContainerService/managedClusters/stop/action
Microsoft.ContainerService/managedClusters/runCommand/action
Microsoft.ContainerService/managedClusters/agentPools/*
Microsoft.Resources/*/read
Microsoft.Resources/tags/write
Microsoft.Authorization/locks/read
Microsoft.Authorization/roleAssignments/read
Microsoft.Authorization/roleDefinitions/read
Microsoft.ManagedIdentity/userAssignedIdentities/assign/action
```

> Note: the [Cast AI changelog (February 2026)](https://docs.cast.ai/changelog/february-2026)
> removed the START/STOP and gallery/snapshot permissions from the *required* list. The baseline
> above is the full advertised list; every variant covers a superset of it, so variants also
> satisfy the reduced list.

## Validity criterion

A configuration is **valid** (and must be accepted by the validation logic) if, for every scope
the official configuration covers, every advertised action above is still allowed. Concretely:

- **Union**: effective permissions at a scope are the union of all roles assigned to that scope
  or to any of its ancestors. A `subscription`-scoped assignment covers all resource group
  scopes.
- **Wildcard matching**: a granted action pattern `G` covers a required action pattern `P` if
  `G` wildcard-matches the string `P` (case-insensitive; `*` matches one or more path segments).
  Examples: `*/read` covers `Microsoft.Compute/*/read`; `Microsoft.Compute/*` covers
  `Microsoft.Compute/virtualMachines/*`; `Microsoft.Network/*` covers
  `Microsoft.Network/networkInterfaces/join/action`.
- **NotActions** subtract from the same role only. A NotAction that matches a required pattern
  invalidates coverage of that pattern by that role.
- **Supersets allowed**: a variant may grant more than the baseline (broader wildcards, wider
  scopes, extra roles) — that does not invalidate it.
- **Deny-list direction**: validity does not require enumerating the required actions. A role may
  grant `*` and rely on `not_actions` alone, as long as no `not_action` pattern matches a required
  action (i.e. every subtracted pattern targets actions outside the advertised set). Built-in
  `Contributor` and the `allow-all-custom-role-with-denylist` variant both use this shape.

## Scope vocabulary

| Scope | Meaning |
| --- | --- |
| `subscription` | `/subscriptions/{subscription_id}` |
| `cluster_resource_group` | Resource group containing the AKS cluster |
| `node_resource_group` | AKS-managed `MC_*` resource group holding the nodes |
| `vnet_resource_group` | Resource group containing the cluster's VNet/subnet when different |

## Entry schema

```jsonc
{
  "id":                       "stable unique identifier of the variant",
  "name":                     "human-readable name",
  "description":              "what the arrangement looks like",
  "rationale":                "why it must still pass validation",
  "expected_valid":           true,   // assertion target for the test
  "required_scope_coverage":  ["cluster_resource_group", "node_resource_group", "vnet_resource_group"],
  "role_definitions": [
    {
      "ref":                "name used by role_assignments",
      "kind":               "custom | builtin",
      "name":               "role display name",
      // custom only:
      "assignable_scopes":  ["scopes the role may be assigned at"],
      // builtin only:
      "role_definition_id": "well-known role definition GUID",
      "actions":            ["granted action patterns"],
      "not_actions":        ["denied action patterns"]
    }
  ],
  "role_assignments": [
    { "role_definition": "<ref>", "scope": "<scope from the vocabulary above>" }
  ]
}
```

Built-in role `actions`/`not_actions` are embedded verbatim from
[Microsoft's built-in role definitions](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)
so a test does not need to fetch them at runtime.

## Curated variants

| # | `id` | Shape |
| --- | --- | --- |
| 1 | `official-module-replica` | Exact official setup: one custom role, all 36 actions, assigned per resource group (baseline) |
| 2 | `official-role-at-subscription-scope` | Same role, single assignment at subscription scope |
| 3 | `per-service-split-custom-roles` | 36 actions partitioned into Compute / Network / control-plane custom roles |
| 4 | `builtin-reader-plus-custom-writer` | Built-in `Reader` at subscription + custom writer role with all non-read actions |
| 5 | `builtin-contributor-at-resource-groups` | Built-in `Contributor` at the three resource groups |
| 6 | `builtin-contributor-at-subscription-scope` | Built-in `Contributor` at subscription |
| 7 | `broadened-wildcard-custom-role` | One role with four service-wide wildcards instead of fine-grained patterns |
| 8 | `wildcard-custom-role-with-notactions` | Same broad role minus unneeded actions via `not_actions` |
| 9 | `custom-reader-at-subscription-custom-writer-at-resource-groups` | Custom read role at subscription, custom write role per resource group |
| 10 | `builtin-roles-mix-with-gapfiller` | `Reader` + `Network Contributor` + `Azure Kubernetes Service Contributor Role` + `Virtual Machine Contributor` + small custom gap-filler |
| 11 | `overlapping-redundant-assignments` | Official role per resource group plus an overlapping broad role at subscription |
| 12 | `allow-all-custom-role-with-denylist` | Custom role granting `*`, restricted only through `not_actions` (role-assignment/policy writes, elevation, host-group/DNS-zone/private-endpoint writes, cluster admin credential listing, subscription lifecycle, …) |

## Consuming from a Go test

```go
import "encoding/json"

type permissionConfiguration struct {
    ID          string `json:"id"`
    ExpectedValid bool `json:"expected_valid"`
    // ... map role_assignments to concrete Azure scope IDs:
    // subscription            -> /subscriptions/<subscription_id>
    // cluster_resource_group  -> /subscriptions/<subscription_id>/resourceGroups/<cluster_rg>
    // node_resource_group     -> /subscriptions/<subscription_id>/resourceGroups/<node_rg>
    // vnet_resource_group     -> /subscriptions/<subscription_id>/resourceGroups/<vnet_rg>
}

func loadPermissionConfigurations(t *testing.T) []permissionConfiguration {
    t.Helper()
    raw, err := os.ReadFile("testdata/aks_permission_configurations.json")
    require.NoError(t, err)
    var configs []permissionConfiguration
    require.NoError(t, json.Unmarshal(raw, &configs))
    return configs
}
```
