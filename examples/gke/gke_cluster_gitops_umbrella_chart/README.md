# GKE + CAST AI GitOps example — umbrella Helm chart

## Overview

This example demonstrates a **GitOps onboarding flow** using the CAST AI umbrella Helm chart (`castai-helm/castai`).
The umbrella chart replaces individual per-component charts and lets you switch between operating modes with a single `helm upgrade` command.

### When is Terraform needed?

| Mode | Terraform required? | What Terraform does |
|---|---|---|
| **Read-only** | No | — |
| **Workload Autoscaler** | No | — |
| **Node Autoscaler / Full** | **Yes** | Creates GCP service account with IAM permissions needed for node provisioning |

> For read-only and workload autoscaler modes you only need a CAST AI API key and Helm. Start there and add Terraform later only if you want node autoscaling.

---

## Umbrella chart modes

The umbrella chart uses **tags** to control which sub-charts are installed.

| Tag | Installed components | Use-case |
|---|---|---|
| `tags.readonly=true` | agent, spot-handler, kvisor, gpu-metrics-exporter | Observe the cluster — no changes made to workloads or nodes |
| `tags.workload-autoscaler=true` | above + cluster-controller, evictor, pod-mutator, workload-autoscaler, workload-autoscaler-exporter | Right-size workload CPU/memory requests automatically |
| `tags.full=true` | all components incl. pod-pinner, live | Full node autoscaler + workload autoscaler |

> Only one tag should be `true` at a time. When upgrading modes use `--reset-then-reuse-values` and flip the tags (see examples below).

---

## Prerequisites

- CAST AI account
- CAST AI **organization member API key** from [console.cast.ai → Service Accounts](https://console.cast.ai/organization/management/access-control/service-accounts)
- `castai-helm` Helm repo:
  ```sh
  helm repo add castai-helm https://castai.github.io/helm-charts
  helm repo update
  ```

---

## Step 1 — Install in read-only mode (Helm only)

No Terraform needed. The API key here is the CAST AI **member** key (not a full-access key).

```sh
helm upgrade -i castai castai-helm/castai -n castai-agent --create-namespace \
  --set global.castai.apiKey="<your-castai-api-key>" \
  --set global.castai.provider="gke" \
  --set tags.readonly=true
```

After the pods become ready your cluster appears as **Read only** in the CAST AI console.
CAST AI can now observe the cluster — no changes are made to your workloads or nodes.

---

## Step 2 (optional) — Upgrade to Workload Autoscaler (Helm only)

When you are ready to let CAST AI right-size CPU/memory requests for your workloads, upgrade the release.
**No Terraform changes required.**

`--reset-then-reuse-values` keeps all previously set values and only applies the overrides you specify.

```sh
helm upgrade castai castai-helm/castai -n castai-agent \
  --reset-then-reuse-values \
  --set tags.readonly=false \
  --set tags.workload-autoscaler=true
```

---

## Step 3 (optional) — Upgrade to Full mode / Node Autoscaler (Terraform + Helm)

Full mode enables node provisioning, bin-packing, spot instance handling, eviction, and pod pinning.
This requires a **GCP service account** with the correct IAM permissions — Terraform creates it.

### 3a. Run Terraform

Fill in your values:

```sh
cp tf.vars.example terraform.tfvars
# edit terraform.tfvars
```

Apply:

```sh
terraform init
terraform apply
```

This registers the cluster with CAST AI and creates the GCP service account.

Capture the outputs — you'll need them to configure the Helm release:

```sh
terraform output cluster_id
terraform output -raw cluster_token
```

> `cluster_token` expires after a few hours if no CAST AI component connects. Run the Helm upgrade promptly after this step.

### 3b. Upgrade the Helm release

If you were already running read-only or workload-autoscaler mode, upgrade using `--reset-then-reuse-values`.
If this is a fresh install, use `helm upgrade -i` and pass `cluster_token` and `cluster_id` from step 3a.

```sh
helm upgrade castai castai-helm/castai -n castai-agent \
  --reset-then-reuse-values \
  --set tags.readonly=false \
  --set tags.workload-autoscaler=false \
  --set tags.full=true
```