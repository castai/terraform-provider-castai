#!/usr/bin/env bash
# check-status.sh — Query CAST AI cluster status via the external-clusters API.
#
# Mirrors the provider's own read path: the terraform-provider-castai uses
# ExternalClusterAPIGetClusterWithResponse(ctx, clusterId) at castai/cluster.go:34
# (delete precheck) and castai/cluster.go:116 (fetchClusterData), which GETs
#   $CASTAI_API_URL/kubernetes/external-clusters/$ID
# and reads JSON200.Status + JSON200.AgentStatus. The SDK struct
# ExternalclusterV1Cluster (castai/sdk/api.gen.go:5652) maps those to JSON fields
#   .status        (string: "ready"|"disconnected"|"deleting"|"archived"|...)
#   .agentStatus    (string: "online"|"disconnected"|"disconnecting"|...)
# This script does the same GET with curl so we can observe cluster state
# without depending on the DAP-launched provider still being alive.
#
# USAGE:
#   check-status.sh [<cluster_id>] [<output_json>]
#
# Arguments:
#   <cluster_id>   CAST AI cluster id. If omitted, resolved in order:
#                  (a) artifacts/cluster-id.txt (written by run-repro.sh),
#                  (b) `terraform output -raw cluster_id` (requires live state).
#   <output_json>  Path to write the status envelope JSON.
#                  Default: artifacts/cluster-status-after.json
#
# OUTPUT:
#   - Stdout: "<status> <agent_status> <cluster_id>" (human-readable, one line).
#   - File at <output_json>: JSON envelope with fields cluster_id, status,
#     agent_status, checked_at (ISO 8601 UTC), http_status, raw (full API JSON
#     object on success, or {"error": "..."} on failure).
#
# EXIT CODES:
#   0  — query succeeded (cluster may or may not be 'ready')
#   1  — precondition error (missing creds/cluster id, curl failure)
#
# ENVIRONMENT:
#   Reads CASTAI_API_URL / CASTAI_API_TOKEN from ~/castai/terraform-provider-castai/.env
#   if not already exported in the environment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPRO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$REPRO_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"
ARTIFACTS_DIR="$REPRO_DIR/artifacts"
CLUSTER_ID_FILE="$ARTIFACTS_DIR/cluster-id.txt"

# --- Load .env (only if vars not already in env) ---
if [[ -z "${CASTAI_API_URL:-}" || -z "${CASTAI_API_TOKEN:-}" ]]; then
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: .env not found at $ENV_FILE and CASTAI_API_URL/TOKEN not in env" >&2
    exit 1
  fi
  # shellcheck disable=SC1090
  set -a; source "$ENV_FILE"; set +a
fi

if [[ -z "${CASTAI_API_URL:-}" || -z "${CASTAI_API_TOKEN:-}" ]]; then
  echo "ERROR: CASTAI_API_URL or CASTAI_API_TOKEN is empty after sourcing $ENV_FILE" >&2
  exit 1
fi

# --- Resolve cluster id ---
CLUSTER_ID="${1:-}"
if [[ -z "$CLUSTER_ID" ]]; then
  if [[ -f "$CLUSTER_ID_FILE" ]]; then
    CLUSTER_ID="$(tr -d '[:space:]' < "$CLUSTER_ID_FILE")"
  elif command -v terraform >/dev/null 2>&1; then
    # Fall back to terraform output (requires provider still registered, but
    # output -raw reads from state, not the provider, so it works post-run).
    CLUSTER_ID="$(cd "$REPRO_DIR" && terraform output -raw cluster_id 2>/dev/null || true)"
  fi
fi

if [[ -z "$CLUSTER_ID" ]]; then
  echo "ERROR: cluster id not provided and not resolvable from $CLUSTER_ID_FILE or terraform output" >&2
  exit 1
fi

# --- Resolve output path ---
OUTPUT_JSON="${2:-$ARTIFACTS_DIR/cluster-status-after.json}"
mkdir -p "$(dirname "$OUTPUT_JSON")"

# --- Query the API ---
# -sS: silent but show errors. -w writes HTTP status. --fail-with-body would mask
# non-2xx bodies; we want the raw body even on 404 (archived clusters).
URL="${CASTAI_API_URL%/}/kubernetes/external-clusters/${CLUSTER_ID}"
TMP_BODY="$(mktemp)"
TMP_CODE="$(mktemp)"
trap 'rm -f "$TMP_BODY" "$TMP_CODE"' EXIT

HTTP_CODE=$(curl -sS \
  -X GET "$URL" \
  -H "X-API-Key: $CASTAI_API_TOKEN" \
  -H "Accept: application/json" \
  -o "$TMP_BODY" \
  -w '%{http_code}' \
  --connect-timeout 10 \
  --max-time 30 \
  2>/dev/null || echo "000")
CURL_RC=$?

if [[ "$CURL_RC" -ne 0 ]]; then
  echo "ERROR: curl failed (rc=$CURL_RC) querying $URL" >&2
  printf '{"cluster_id":"%s","status":null,"agent_status":null,"checked_at":"%s","http_status":"%s","raw":{"error":"curl rc=%s"}}\n' \
    "$CLUSTER_ID" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$HTTP_CODE" "$CURL_RC" > "$OUTPUT_JSON"
  exit 1
fi

BODY="$(cat "$TMP_BODY")"
CHECKED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --- Parse status + agentStatus (defensive: field may be null on 404/archived) ---
STATUS="$(jq -r '.status // "null"' <<<"$BODY" 2>/dev/null || echo "null")"
AGENT_STATUS="$(jq -r '.agentStatus // "null"' <<<"$BODY" 2>/dev/null || echo "null")"

# Some endpoints return an error object instead of a cluster on non-2xx; surface
# the http code in the envelope so the bug-repro check (status != 'ready') is
# distinguishable from a transport error.
printf '{"cluster_id":"%s","status":%s,"agent_status":%s,"checked_at":"%s","http_status":%s,"raw":%s}\n' \
  "$CLUSTER_ID" \
  "$(jq 'if .status then .status else null end' <<<"$BODY" 2>/dev/null || echo 'null')" \
  "$(jq 'if .agentStatus then .agentStatus else null end' <<<"$BODY" 2>/dev/null || echo 'null')" \
  "$CHECKED_AT" \
  "$HTTP_CODE" \
  "$(jq '.' <<<"$BODY" 2>/dev/null || echo "{\"error\":\"non-json body\",\"body\":$(jq -Rs . <<<"$BODY")}")" \
  > "$OUTPUT_JSON"

# --- Human-readable stdout ---
printf '%s %s %s (http %s)\n' "$STATUS" "$AGENT_STATUS" "$CLUSTER_ID" "$HTTP_CODE"
