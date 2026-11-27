#!/usr/bin/env sh
# Prove that CAST AI traffic from this cluster takes the Private Link path.
# !!!!
# THIS SCRIPT IS FOR TESTING PURPOSES ONLY.
# It is not required for using private link at all.
# !!!!
# Usage:
#   ./scripts/verify.sh [--context NAME] [--namespace NAME]
#
# Environment:
#   KUBE_CONTEXT   same as --context
#   PE_IP          expected private endpoint IP (default: read from terraform output)
#   KEEP_POD=1     keep the verification pod and namespace when done
#
# Every check prints PASS, FAIL or SKIP, and the script exits non-zero if at least
# one check fails. Nothing outside the throwaway namespace is touched, and the
# kube-context is only ever passed explicitly - never switched.
set -eu

CONTEXT="${KUBE_CONTEXT:-}"
NAMESPACE="castai-pls-verify"
POD="pls-verify"
IMAGE="${IMAGE:-ghcr.io/nicolaka/netshoot:v0.16}"
PASSED=0
FAILED=0
SKIPPED=0
NAMESPACE_CREATED=0

usage() {
  cat <<'EOF'
Usage: ./scripts/verify.sh [--context NAME] [--namespace NAME]

  --context NAME   kube-context of the cluster to verify (default: current context)
  --namespace NAME throwaway namespace for the check pod (default: castai-pls-verify)

Environment:
  KUBE_CONTEXT     same as --context
  PE_IP            expected private endpoint IP (default: terraform output)
  HOSTS            hostnames to check (default: terraform output castai_private_dns_fqdns,
                   falling back to api.cast.ai grpc.cast.ai)
  KEEP_POD=1       keep the check pod and namespace after the run
  IMAGE            check pod image (default: ghcr.io/nicolaka/netshoot:v0.16)

Checks, all from inside the cluster:
  1. every CAST AI hostname (from terraform output, or HOSTS="h1 h2")
     resolves to the private endpoint
  2. https://<api host>/ used the private endpoint (remote_ip)
  3. the certificate served through the endpoint is the real api host one
  4. (on the host, if dig exists) public DNS returns a different address
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --context)
      CONTEXT="${2:?--context requires a value}"
      shift 2
      ;;
    --context=*)
      CONTEXT="${1#*=}"
      shift
      ;;
    --namespace)
      NAMESPACE="${2:?--namespace requires a value}"
      shift 2
      ;;
    --namespace=*)
      NAMESPACE="${1#*=}"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

CTX=""
if [ -n "$CONTEXT" ]; then
  CTX="--context=$CONTEXT"
fi

kubectl_() { kubectl $CTX "$@"; }
pexec() { kubectl_ -n "$NAMESPACE" exec "$POD" -- "$@"; }

record() {
  case "$1" in
    PASS)
      PASSED=$((PASSED + 1))
      printf 'PASS  %s\n' "$2"
      ;;
    FAIL)
      FAILED=$((FAILED + 1))
      printf 'FAIL  %s\n' "$2"
      ;;
    SKIP)
      SKIPPED=$((SKIPPED + 1))
      printf 'SKIP  %s\n' "$2"
      ;;
  esac
}

cleanup() {
  if [ "${KEEP_POD:-0}" = "1" ]; then
    echo "KEEP_POD=1: leaving pod $POD in namespace $NAMESPACE"
  else
    kubectl_ -n "$NAMESPACE" delete pod "$POD" --ignore-not-found --wait=false >/dev/null 2>&1 || true
    if [ "$NAMESPACE_CREATED" = "1" ]; then
      kubectl_ delete namespace "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
    fi
  fi
  :
}
trap cleanup EXIT INT TERM

# --- expected private endpoint IP --------------------------------------------
if [ -z "${PE_IP:-}" ]; then
  for bin in tofu terraform; do
    if command -v "$bin" >/dev/null 2>&1; then
      PE_IP=$(cd "$ROOT_DIR" && "$bin" output -raw private_endpoint_ip 2>/dev/null || true)
      if [ -n "$PE_IP" ]; then
        break
      fi
    fi
  done
fi

# --- throwaway pod -----------------------------------------------------------
echo "==> starting $IMAGE pod in namespace $NAMESPACE${CONTEXT:+ (context $CONTEXT)}"
if ! kubectl_ get namespace "$NAMESPACE" >/dev/null 2>&1; then
  kubectl_ create namespace "$NAMESPACE" >/dev/null
  NAMESPACE_CREATED=1
fi
kubectl_ -n "$NAMESPACE" delete pod "$POD" --ignore-not-found --wait=true >/dev/null 2>&1 || true
kubectl_ -n "$NAMESPACE" run "$POD" --image="$IMAGE" --restart=Never --command -- sleep 900 >/dev/null
if ! kubectl_ -n "$NAMESPACE" wait --for=condition=Ready "pod/$POD" --timeout=180s >/dev/null; then
  echo "FAIL  the check pod did not become Ready" >&2
  kubectl_ -n "$NAMESPACE" describe pod "$POD" | tail -20 >&2 || true
  exit 1
fi

# Fall back to whatever the cluster resolves, if Terraform output is unavailable.
if [ -z "${PE_IP:-}" ]; then
  PE_IP=$(pexec sh -c 'getent hosts api.cast.ai 2>/dev/null | head -1 | cut -d" " -f1' || true)
fi
if [ -z "${PE_IP:-}" ]; then
  record FAIL "could not determine the private endpoint IP (pass PE_IP=... or run this next to the Terraform state)"
  echo
  printf '%d passed, %d failed, %d skipped\n' "$PASSED" "$FAILED" "$SKIPPED"
  exit 1
fi
echo "==> expecting private endpoint IP: $PE_IP"

# --- hostnames to check ------------------------------------------------------
if [ -z "${HOSTS:-}" ]; then
  for bin in tofu terraform; do
    if command -v "$bin" >/dev/null 2>&1; then
      FQDNS=$(cd "$ROOT_DIR" && "$bin" output -json castai_private_dns_fqdns 2>/dev/null | tr -d '"[]' | tr ',' ' ' | tr -s ' \n' ' ' || true)
      if [ -n "$(printf '%s' "${FQDNS:-}" | tr -d '[:space:]')" ]; then
        break
      fi
      FQDNS=""
    fi
  done
  HOSTS=${FQDNS:-"api.cast.ai grpc.cast.ai"}
fi

API_HOST=$(printf '%s\n' $HOSTS | grep -E '(^|\.)api\.' | head -1 || true)
API_HOST=${API_HOST:-api.cast.ai}

echo "==> checking hostnames: $HOSTS"

# --- 1. CAST AI hostnames resolve to the endpoint ----------------------------
for host in $HOSTS; do
  resolved=$(pexec sh -c "getent hosts $host 2>/dev/null | head -1 | cut -d' ' -f1" || true)
  if [ "$resolved" = "$PE_IP" ]; then
    record PASS "$host resolves to $resolved (private endpoint)"
  else
    record FAIL "$host resolves to '${resolved:-nothing}', expected $PE_IP - wrong kube-context, or the private zone is not linked to this VNet"
  fi
done

# --- 2. request takes the private path ---------------------------------------
curl_out=$(pexec sh -c "curl -sS --max-time 15 --connect-timeout 5 -o /dev/null -w '%{remote_ip}' https://$API_HOST/" 2>&1) ||
  curl_out="error"
if [ "$curl_out" = "$PE_IP" ]; then
  record PASS "https://$API_HOST/ used the private endpoint (remote_ip=$curl_out)"
else
  record FAIL "https://$API_HOST/ did not use the private endpoint (got '${curl_out}', expected $PE_IP)"
fi

# --- 3. real upstream certificate through the endpoint -----------------------
subject=$(pexec sh -c "echo | openssl s_client -connect $PE_IP:443 -servername $API_HOST 2>/dev/null | openssl x509 -noout -subject 2>/dev/null" || true)
case "$subject" in
  *"$API_HOST"*)
    record PASS "TLS through the endpoint serves the real $API_HOST certificate"
    ;;
  *)
    record FAIL "no $API_HOST certificate through $PE_IP:443 (SNI allow-list mismatch, or the endpoint is not approved)"
    ;;
esac

# --- 4. public DNS still differs --------------------------------------------
if command -v dig >/dev/null 2>&1; then
  public_ip=$(dig +short "$API_HOST" | head -1)
  if [ -n "$public_ip" ] && [ "$public_ip" != "$PE_IP" ]; then
    record PASS "public DNS still returns $public_ip, so the private answer is an override"
  else
    record SKIP "public DNS returned '${public_ip:-nothing}'"
  fi
else
  record SKIP "dig not installed on the host"
fi

# --- summary -----------------------------------------------------------------
echo
printf '%d passed, %d failed, %d skipped\n' "$PASSED" "$FAILED" "$SKIPPED"
[ "$FAILED" -eq 0 ]
