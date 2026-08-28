#!/usr/bin/env bash
# run-repro.sh — Terraform driver for the castai_gke_cluster taint→reapply bug repro.
#
# Drives the Terraform sequence against the real GKE cluster furkhatk-2007:
#   init → apply (initial create) → taint → apply (taint-recreate, TF_LOG=DEBUG)
#
# PROVIDER LAUNCH (NOT done by this script):
#   The CAST AI provider must ALREADY be running under the kimchi DAP debugger
#   (dlv adapter, launch mode, args=["--debug"]). The orchestrator (kimchi agent)
#   calls debug_launch, then debug_continue to let the provider print its
#   TF_REATTACH_PROVIDERS reattach config to stdout. The orchestrator captures
#   that line, exports TF_REATTACH_PROVIDERS, sets breakpoints via
#   debug_set_breakpoint, then invokes THIS SCRIPT (typically in background)
#   so it can poll DAP stops and capture state/eval snapshots while Terraform
#   drives the provider through the Delete→Create flow.
#
# PRECONDITIONS:
#   - TF_REATTACH_PROVIDERS env var set (by orchestrator, from DAP provider stdout)
#   - ~/castai/terraform-provider-castai/.env exists (exports CASTAI_API_URL/TOKEN)
#   - terraform + jq binaries on PATH
#
# ARTIFACTS (written to ./artifacts/):
#   init.log             — terraform init output
#   apply-create.log     — terraform apply output (initial create)
#   taint.log            — terraform taint output (or -replace fallback plan)
#   apply-recreate.log   — terraform apply output (taint-recreate)
#   provider.log         — TF_LOG=DEBUG provider log (recreate phase: Delete + Create)
#   cluster-id.txt       — cluster id parsed from terraform.tfstate
#   run-summary.txt      — tab-separated: timestamp \t phase \t result
#
# USAGE:
#   # After orchestrator launched provider under DAP + exported TF_REATTACH_PROVIDERS:
#   TF_REATTACH_PROVIDERS='{...}' bash run-repro.sh > artifacts/run-repro.stdout 2>&1 &
#
# EXIT CODES:
#   0  — all phases succeeded
#   1  — a terraform phase failed (see corresponding artifacts/*.log)
#   2  — precondition not met (TF_REATTACH_PROVIDERS unset or .env missing)

set -euo pipefail

# --- Paths (resolved relative to this script so it runs from anywhere) ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ARTIFACTS_DIR="$SCRIPT_DIR/artifacts"
ENV_FILE="$REPO_ROOT/.env"
STATE_FILE="$SCRIPT_DIR/terraform.tfstate"

mkdir -p "$ARTIFACTS_DIR"

# --- Precondition: TF_REATTACH_PROVIDERS ---
if [[ -z "${TF_REATTACH_PROVIDERS:-}" ]]; then
  echo "ERROR: TF_REATTACH_PROVIDERS is not set." >&2
  echo "       Launch the provider under the kimchi DAP debugger first (dlv," >&2
  echo "       launch mode, args=[\"--debug\"]); the orchestrator captures" >&2
  echo "       TF_REATTACH_PROVIDERS from the provider's stdout and exports it" >&2
  echo "       before invoking this script." >&2
  exit 2
fi

# --- Precondition: .env ---
if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: .env not found at $ENV_FILE" >&2
  exit 2
fi

# --- Load CAST AI credentials from .env (uses `export VAR=value` lines) ---
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

# Verify credentials loaded.
if [[ -z "${CASTAI_API_URL:-}" || -z "${CASTAI_API_TOKEN:-}" ]]; then
  echo "ERROR: CASTAI_API_URL or CASTAI_API_TOKEN is empty after sourcing $ENV_FILE" >&2
  exit 2
fi

# Export TF_VAR_* mirrors for the variables declared in main.tf, and ensure the
# bare env vars are exported (provider's EnvDefaultFunc reads CASTAI_API_URL/TOKEN).
export TF_VAR_castai_api_url="$CASTAI_API_URL"
export TF_VAR_castai_api_token="$CASTAI_API_TOKEN"
export CASTAI_API_URL CASTAI_API_TOKEN

cd "$SCRIPT_DIR"

# --- Helpers ---
log() { printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }
status_line() {
  local phase="$1" result="$2"
  printf '%s\t%s\t%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$phase" "$result" \
    >> "$ARTIFACTS_DIR/run-summary.txt"
}

: > "$ARTIFACTS_DIR/run-summary.txt"

log "=== run-repro.sh started ==="
log "working dir: $SCRIPT_DIR"
log "TF_REATTACH_PROVIDERS length: ${#TF_REATTACH_PROVIDERS}"
log "CASTAI_API_URL: $CASTAI_API_URL"

# --- Phase 0: terraform init ---
log "=== Phase 0: terraform init ==="
if terraform init -input=false -no-color > "$ARTIFACTS_DIR/init.log" 2>&1; then
  status_line "init" "ok"
  log "init: ok"
else
  status_line "init" "fail"
  log "init: FAIL (see artifacts/init.log)"
  exit 1
fi

# --- Phase 1: initial create ---
# Registers the existing GKE cluster with CAST AI (ExternalClusterAPIRegisterCluster).
# Breakpoints in resourceCastaiGKEClusterCreate will fire here under DAP.
log "=== Phase 1: terraform apply (initial create) ==="
if terraform apply -auto-approve -input=false -no-color \
    > "$ARTIFACTS_DIR/apply-create.log" 2>&1; then
  status_line "apply-create" "ok"
  log "apply-create: ok"
else
  rc=$?
  status_line "apply-create" "fail(rc=$rc)"
  log "apply-create: FAIL rc=$rc (see artifacts/apply-create.log)"
  exit 1
fi

# --- Phase 1.5: taint the resource ---
# terraform taint is deprecated since 0.15.2 in favor of `terraform apply -replace=`,
# but still functional in 1.x. The success criteria explicitly require `terraform taint`.
# If a future terraform removes the command, fall back to `plan -replace` + `apply tfplan`.
log "=== Phase 1.5: terraform taint castai_gke_cluster.this ==="
USE_PLAN=0
if terraform taint castai_gke_cluster.this > "$ARTIFACTS_DIR/taint.log" 2>&1; then
  status_line "taint" "ok"
  log "taint: ok"
else
  rc=$?
  status_line "taint" "fail(rc=$rc)"
  log "taint: FAIL rc=$rc — trying -replace fallback (see artifacts/taint.log)"
  echo "--- fallback: terraform plan -replace=castai_gke_cluster.this ---" \
    >> "$ARTIFACTS_DIR/taint.log"
  if terraform plan -replace=castai_gke_cluster.this -out=tfplan \
      -input=false -no-color >> "$ARTIFACTS_DIR/taint.log" 2>&1; then
    status_line "taint-fallback-plan" "ok"
    log "taint-fallback: replacement plan written to tfplan"
    USE_PLAN=1
  else
    rc2=$?
    status_line "taint-fallback-plan" "fail(rc=$rc2)"
    log "taint-fallback: FAIL rc=$rc2"
    exit 1
  fi
fi

# --- Phase 2: taint-recreate apply (with TF_LOG=DEBUG) ---
# This is the phase that reproduces the bug: terraform destroys (Delete:
# resourceCastaiGKEClusterDelete → resourceCastaiClusterDelete → triggerDisconnect
# then triggerDelete) and recreates (Create: resourceCastaiGKEClusterCreate).
# Breakpoints fire in BOTH branches under DAP. TF_LOG=DEBUG + TF_LOG_PATH capture
# the provider log for grep verification ('Cluster with id', 'Deleting cluster').
log "=== Phase 2: terraform apply (taint-recreate) — TF_LOG=DEBUG ==="
export TF_LOG=DEBUG
export TF_LOG_PATH="$ARTIFACTS_DIR/provider.log"
: > "$TF_LOG_PATH"  # truncate provider log at start of recreate phase

if [[ "$USE_PLAN" == "1" ]]; then
  APPLY_ARGS=("tfplan")
else
  APPLY_ARGS=("-auto-approve" "-input=false" "-no-color")
fi

if terraform apply "${APPLY_ARGS[@]}" > "$ARTIFACTS_DIR/apply-recreate.log" 2>&1; then
  status_line "apply-recreate" "ok"
  log "apply-recreate: ok"
else
  rc=$?
  status_line "apply-recreate" "fail(rc=$rc)"
  log "apply-recreate: FAIL rc=$rc (see artifacts/apply-recreate.log)"
  exit 1
fi

# --- Extract cluster id from terraform.tfstate (jq, no provider connection needed) ---
# We parse the state file directly to avoid depending on the DAP provider still
# being alive for `terraform output`/`terraform show` (which may re-query schema).
CLUSTER_ID=""
if [[ -f "$STATE_FILE" ]]; then
  CLUSTER_ID="$(jq -r '
    .resources[]
    | select(.type == "castai_gke_cluster" and .name == "this")
    | .instances[0].attributes.id // empty
  ' "$STATE_FILE" 2>/dev/null || true)"
fi

if [[ -z "$CLUSTER_ID" ]]; then
  log "WARN: could not extract cluster id from terraform.tfstate"
  status_line "extract-cluster-id" "empty"
else
  printf '%s\n' "$CLUSTER_ID" > "$ARTIFACTS_DIR/cluster-id.txt"
  status_line "extract-cluster-id" "ok($CLUSTER_ID)"
  log "cluster-id: $CLUSTER_ID"
fi

log "=== run-repro.sh complete ==="
log "artifacts: $ARTIFACTS_DIR/ (init.log, apply-create.log, taint.log, apply-recreate.log, provider.log, cluster-id.txt, run-summary.txt)"
status_line "run-complete" "ok"
