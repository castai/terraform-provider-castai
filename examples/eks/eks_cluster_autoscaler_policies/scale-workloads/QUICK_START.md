# Quick Start Guide

## Setup (One Time)

```bash
cd scale-workloads

# Deploy all workloads
kubectl apply -f .

# Make script executable (if not already)
chmod +x scale.sh
```

## Monarch Workload Overview

These workloads simulate Spark job phases with different resource requirements:
- **master**: Driver pod running on on-demand nodes (small resource requirements)
- **prep**: PREP phase workers on spot nodes (high memory, moderate CPU)
- **mig**: MIG phase workers on spot nodes (very high CPU and memory)
- **accumall**: ACCUMALL phase workers on spot nodes (high memory, moderate CPU)

All workloads target nodes with labels:
- `spark-nodeselect-instance-type: amd64-64-16`
- `spark-nodeselect-nodepool-group: hyper`
- `spark-nodeselect-preemptible: "true"` (for prep/mig/accumall) or `"false"` (for master)

## Common Commands

### Scale Workloads by Replicas

```bash
# Scale master to 10 replicas (on-demand nodes)
./scale.sh --size master --replicas 10

# Scale prep workers to 100 replicas (spot nodes)
./scale.sh --size prep --replicas 100

# Scale mig workers to 50 replicas (requires large nodes)
./scale.sh --size mig --replicas 50

# Scale in a specific namespace
./scale.sh --size prep --replicas 100 --namespace production
```

### Scale to Target Node Counts

```bash
# Scale prep to target ~100 nodes
./scale.sh --size prep --nodes 100

# Scale mig to target ~50 nodes (very high CPU requirements)
./scale.sh --size mig --nodes 50

# Scale accumall to target ~100 nodes
./scale.sh --size accumall --nodes 100

# Scale master for ~10 on-demand nodes
./scale.sh --size master --nodes 10
```

### Scale Down

```bash
# Scale down everything
./scale.sh --size all --replicas 0

# Scale down individual workload
./scale.sh --size mig --replicas 0
```

### Check Status

```bash
# View deployment status
kubectl get deployments -l app=monarch

# View deployments in a specific namespace
kubectl get deployments -l app=monarch -n production

# Count running pods
kubectl get pods -l app=monarch --field-selector=status.phase=Running | wc -l

# Watch nodes
kubectl get nodes -w

# Check pending pods
kubectl get pods -l app=monarch --field-selector=status.phase=Pending

# Check which nodes the pods are running on
kubectl get pods -l app=monarch -o wide
```

## Resource Profiles

| Workload  | CPU      | Memory   | Ephemeral Storage | Pods per 64-vCPU Node | Node Type  |
|-----------|----------|----------|-------------------|----------------------|------------|
| master    | 500m     | 1Gi      | 4Mi              | ~100+                | on-demand  |
| prep      | 12800m   | 44800Mi  | 4Mi              | ~5                   | spot       |
| mig       | 25600m   | 44800Mi  | 4Mi              | ~2                   | spot       |
| accumall  | 12800m   | 44800Mi  | 4Mi              | ~5                   | spot       |

## Important Notes

- **prep, mig, accumall require large instance types**: Recommend 64+ vCPU nodes (e.g., c5.18xlarge, c6i.32xlarge)
- **mig has extremely high CPU requirements**: 25.6 cores per pod, needs very large nodes
- The `--nodes` calculation assumes 64 vCPU nodes for prep/mig/accumall
- Actual pod density varies significantly with instance type
- Always verify your node configuration supports the requested labels/taints

## Troubleshooting

### "error: no objects passed to scale"

This error means the deployment doesn't exist. The script now checks for this and will show available deployments. To fix:

```bash
# Check what namespace your deployments are in
kubectl get deployments --all-namespaces | grep monarch

# Use the correct namespace
./scale.sh --size prep --replicas 12 --namespace <your-namespace>

# Or verify the deployment name pattern matches "monarch-<workload>"
kubectl get deployments
```

### Deployments Not Found

If `kubectl apply -f .` was successful but deployments aren't found:
- Verify you're in the correct kubectl context: `kubectl config current-context`
- Check if deployments are in a different namespace: `kubectl get deployments -A`
- Ensure the YAML files were applied: `kubectl get deployments`

## Tips

- Start with small replica counts to verify node provisioning works correctly
- Monitor costs during testing - large instances are expensive
- Always clean up after testing: `./scale.sh --size all --replicas 0`
- Check autoscaler logs: `kubectl logs -n kube-system -l app=cluster-autoscaler -f`
- Verify node labels match workload requirements: `kubectl get nodes --show-labels | grep spark-nodeselect`
- If you get "deployment not found" errors, the script will list available deployments to help debug

## Example Test Scenario

```bash
# 1. Deploy master on on-demand nodes
./scale.sh --size master --replicas 1

# 2. Wait for master to be running
kubectl get pods -l app=monarch-master -w

# 3. Start prep phase
./scale.sh --size prep --nodes 10

# 4. Wait for prep to complete (simulate)
sleep 60

# 5. Start mig phase
./scale.sh --size mig --nodes 5

# 6. Clean up everything
./scale.sh --size all --replicas 0

# Note: Add --namespace <name> to any command to target a specific namespace
```
