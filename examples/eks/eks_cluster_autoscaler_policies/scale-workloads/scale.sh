#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print usage
usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Scale Monarch Spark workloads to control cluster size.

OPTIONS:
    -s, --size SIZE         Workload type: master, prep, mig, accumall, or all
    -r, --replicas COUNT    Number of replicas (default: 0)
    -n, --nodes TARGET      Target number of nodes (estimates replicas based on workload)
    -N, --namespace NS      Kubernetes namespace (default: current context)
    -h, --help              Show this help message

EXAMPLES:
    # Scale master workload to 10 replicas
    $0 --size master --replicas 10

    # Scale all workloads to 0 (downscale cluster)
    $0 --size all --replicas 0

    # Scale prep workload to target ~100 nodes
    $0 --size prep --nodes 100

    # Scale mig workload to target ~50 nodes (high CPU) in specific namespace
    $0 --size mig --nodes 50 --namespace production

WORKLOAD TYPES AND RESOURCE REQUESTS:
    master:   500m CPU, 1Gi RAM       (~15 pods per 8-vCPU node)    [on-demand]
    prep:     12800m CPU, 44800Mi RAM (~0.6 pods per 8-vCPU node)   [spot, requires 16+ vCPU nodes]
    mig:      25600m CPU, 44800Mi RAM (~0.3 pods per 8-vCPU node)   [spot, requires 32+ vCPU nodes]
    accumall: 12800m CPU, 44800Mi RAM (~0.6 pods per 8-vCPU node)   [spot, requires 16+ vCPU nodes]

NOTE: prep, mig, and accumall require large instance types (64 vCPU recommended).
      Actual pods per node varies greatly with instance type.
EOF
  exit 0
}

# Function to calculate replicas based on target nodes
calculate_replicas() {
  local size=$1
  local target_nodes=$2
  local pods_per_node

  case $size in
  master)
    pods_per_node=15
    ;;
  prep)
    # Assumes 64 vCPU nodes, ~5 pods per node
    pods_per_node=5
    ;;
  mig)
    # Assumes 64 vCPU nodes, ~2 pods per node
    pods_per_node=2
    ;;
  accumall)
    # Assumes 64 vCPU nodes, ~5 pods per node
    pods_per_node=5
    ;;
  *)
    echo -e "${RED}Invalid workload type: $size${NC}"
    echo -e "${YELLOW}Valid types: master, prep, mig, accumall, all${NC}"
    exit 1
    ;;
  esac

  echo $((target_nodes * pods_per_node))
}

# Function to scale deployment
scale_deployment() {
  local workload=$1
  local replicas=$2
  local deployment="monarch-$workload"
  local kubectl_args=""

  if [ -n "$NAMESPACE" ]; then
    kubectl_args="-n $NAMESPACE"
  fi

  echo -e "${YELLOW}Scaling deployment '$deployment' to $replicas replicas...${NC}"

  # Check if deployment exists first
  if ! kubectl get deployment "$deployment" $kubectl_args &>/dev/null; then
    echo -e "${RED}✗ Deployment '$deployment' not found${NC}"
    echo -e "${YELLOW}Available deployments:${NC}"
    kubectl get deployments $kubectl_args 2>/dev/null || echo "  No deployments found"
    exit 1
  fi

  kubectl scale deployment "$deployment" --replicas="$replicas" $kubectl_args

  if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully scaled $deployment${NC}"
  else
    echo -e "${RED}✗ Failed to scale $deployment${NC}"
    exit 1
  fi
}

# Function to get current deployment status
get_status() {
  local kubectl_args=""
  if [ -n "$NAMESPACE" ]; then
    kubectl_args="-n $NAMESPACE"
  fi

  echo -e "\n${YELLOW}Current Deployment Status:${NC}"
  kubectl get deployments -l app=monarch -o wide $kubectl_args
  echo ""
  kubectl get pods -l app=monarch --field-selector=status.phase=Running $kubectl_args | head -20
  local total_pods=$(kubectl get pods -l app=monarch --field-selector=status.phase=Running --no-headers $kubectl_args 2>/dev/null | wc -l | tr -d ' ')
  echo -e "\n${GREEN}Total Running Pods: $total_pods${NC}"
  echo -e "${YELLOW}Total Nodes: $(kubectl get nodes --no-headers 2>/dev/null | wc -l | tr -d ' ')${NC}"
}

# Parse arguments
SIZE=""
REPLICAS=""
TARGET_NODES=""
NAMESPACE="default"

while [[ $# -gt 0 ]]; do
  case $1 in
  -s | --size)
    SIZE="$2"
    shift 2
    ;;
  -r | --replicas)
    REPLICAS="$2"
    shift 2
    ;;
  -n | --nodes)
    TARGET_NODES="$2"
    shift 2
    ;;
  -N | --namespace)
    NAMESPACE="$2"
    shift 2
    ;;
  -h | --help)
    usage
    ;;
  *)
    echo -e "${RED}Unknown option: $1${NC}"
    usage
    ;;
  esac
done

# Validate inputs
if [ -z "$SIZE" ]; then
  echo -e "${RED}Error: Size is required${NC}"
  usage
fi

if [ -z "$REPLICAS" ] && [ -z "$TARGET_NODES" ]; then
  echo -e "${RED}Error: Either --replicas or --nodes must be specified${NC}"
  usage
fi

# Calculate replicas from target nodes if specified
if [ -n "$TARGET_NODES" ]; then
  if [ "$SIZE" == "all" ]; then
    echo -e "${RED}Error: Cannot use --nodes with --size all. Please specify a specific size.${NC}"
    exit 1
  fi
  REPLICAS=$(calculate_replicas "$SIZE" "$TARGET_NODES")
  echo -e "${YELLOW}Targeting $TARGET_NODES nodes with $SIZE pods = $REPLICAS replicas${NC}"
fi

# Scale deployments
if [ "$SIZE" == "all" ]; then
  scale_deployment "master" "$REPLICAS"
  scale_deployment "prep" "$REPLICAS"
  scale_deployment "mig" "$REPLICAS"
  scale_deployment "accumall" "$REPLICAS"
else
  scale_deployment "$SIZE" "$REPLICAS"
fi

# Show status
get_status

echo -e "\n${GREEN}Done! Monitor cluster autoscaling with: kubectl get nodes -w${NC}"
