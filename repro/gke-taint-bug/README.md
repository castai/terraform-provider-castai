# GKE taint→reapply bug reproduction

Minimal reproduction of the `castai_gke_cluster` taint→reapply bug: `terraform apply`
reports a successful recreate, but the cluster ends up disconnected/deleted in CAST AI.

> This is a skeleton. Full detail (bug summary, step-by-step repro, debugger
> walkthrough, artifact index, teardown procedure) is filled in during the final
> phase of the ferment. See the section stubs below.

## Bug

_Summary of the taint→reapply bug — to be filled in Phase 3._

Symptom: after `terraform taint castai_gke_cluster.this` followed by
`terraform apply -auto-approve`, Terraform reports `Apply complete!` but the CAST AI
cluster is observed in a non-`ready` state (`disconnected` / `deleted` / `archived`).

## Reproduce

_How to run the reproduction — to be filled in Phase 3._

Uses `run-repro.sh`, which launches the locally-built debug provider under dlv,
parses `TF_REATTACH_PROVIDERS` from provider stdout, then runs
`init` → `apply` (create) → `taint` → `apply` (recreate).

## Debug

_How the DAP capture works — to be filled in Phase 3._

Breakpoints are set in `castai/resource_gke_cluster.go` and `castai/cluster.go` on
the Delete/Create code paths; `dap-state-*.json` captures runtime state and
`api-calls.jsonl` captures SDK `ExternalClusterAPI*WithResponse` call/response pairs
via DAP eval at the call sites.

## Artifacts

_Artifact layout under `artifacts/` — to be filled in Phase 3._

| File | Purpose |
| --- | --- |
| `provider.log` | `TF_LOG=DEBUG` provider logs |
| `dap-state-*.json` | DAP-captured runtime state at each breakpoint hit |
| `api-calls.jsonl` | API call/response records captured via DAP eval |
| `cluster-timeline.jsonl` | CAST AI cluster status polled every 10s |
| `cluster-status-after.json` | Final cluster status (proves bug reproduction) |

## Teardown

_How to clean up — to be filled in Phase 3._

Run `teardown.sh` (NOT auto-run by the repro; the cluster is left in the buggy
state for manual CAST AI UI inspection).
