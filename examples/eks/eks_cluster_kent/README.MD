# EKS cluster with Kent (Kentroller)

Onboards an EKS cluster to CAST AI with Kent (Kentroller) enabled. Kentroller
runs in-cluster, maintains a streaming `StreamActions` connection to the Kent
backend, and reconciles `RebalancePlan` CRDs into Karpenter `NodeClaim`s.

## What this example installs

- **VPC** + **EKS v21** cluster (public endpoint access)
- A **managed node group** dedicated to the Karpenter controller — tainted
  `karpenter.sh/controller=true:NoSchedule` and labeled
  `karpenter.sh/controller=true`. CoreDNS tolerates this taint so it can run
  before Karpenter exists (avoids bootstrap deadlock). The MNG launch template
  sets `httpPutResponseHopLimit = 2` for CAST AI compatibility.
- **Karpenter 1.7.x** + one default `EC2NodeClass` and one default `NodePool`
  (instance categories `c`/`m`/`r`, generation > 1, nitro, amd64; `WhenEmpty`
  consolidation after 30s).
- A dedicated **`<cluster_name>-castai-workload` IAM role** with a narrow
  inline policy (`ec2:Describe*` for inventory, `pricing:GetProducts`,
  `eks:DescribeCluster`), and **EKS Pod Identity associations** binding it
  to the `castai-agent` and `castai-cluster-controller` ServiceAccounts so
  those in-cluster workloads can call AWS APIs during cluster registration.
- **CAST AI umbrella chart** with `kent.enabled=true`, which installs the
  Kent subchart bundle:
  - `castai-agent`
  - `castai-cluster-controller`
  - `castai-kentroller`
  - `castai-workload-autoscaler` (+ exporter)
  - `castai-live` — CLM, controller scaled to 0 by default (installed-but-dormant)
  - `castai-pod-mutator`
  - `castai-spot-handler`
  - `castai-kvisor` (controller-only)

## Prerequisites

- AWS credentials with EKS / IAM / VPC create rights
- `aws` CLI on `PATH` (used by the kubernetes / helm / kubectl providers'
  `exec` blocks for cluster auth)
- A CAST AI organization API token (console.cast.ai → Settings → API access)

### Important — KENT-435 token-permission requirement

Your CAST AI environment must include the KENT-435 token-permission fix
(kubecast MR !44713). If your token was issued **before** that change rolled
out to your environment, regenerate it after the upgrade — older tokens are
missing the `StreamActions`, `AwsDataProxy`, and `SpotInterruptionPredictor`
policies, and Kentroller will fail to connect with `PermissionDenied`.

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars
# fill in cluster_name, cluster_region, castai_api_token
terraform init
terraform apply
```

A single `terraform apply` brings up the full stack. Slice ordering (EKS →
Karpenter → CAST AI umbrella) is enforced by `depends_on` in `castai.tf`, so
the `kent-preflight` pre-install Job sees Karpenter before the umbrella tries
to install.

## Verify

| Check                      | Command                                                                              |
|----------------------------|--------------------------------------------------------------------------------------|
| Karpenter running          | `kubectl -n karpenter get pods`                                                      |
| Karpenter CRs applied      | `kubectl get nodepools.karpenter.sh,ec2nodeclasses.karpenter.k8s.aws`                |
| All CAST AI pods Running   | `kubectl -n castai-agent get pods`                                                   |
| Cluster registered         | `kubectl -n castai-agent get cm castai-agent-metadata -o jsonpath='{.data.CLUSTER_ID}'` |
| Stream connected           | `kubectl -n castai-agent logs deploy/castai-kentroller \| grep -E 'stream connected\|stream registered'` |
| Cluster visible in CAST AI | https://console.cast.ai → your cluster, Kent indicator on                            |

A working stream looks like (from a fresh apply):

```
msg="getStream: stream registered confirmation received, stream is ready" cluster_id=<uuid>
msg="streaming messages..." cluster_id=<uuid>
msg="stream connected, syncing all schedules to Kent" cluster_id=<uuid>
```

No `PermissionDenied` errors should appear in kentroller logs.

## Troubleshooting

### `castai-agent` in `CrashLoopBackOff` with `DescribeInstances` errors

Symptom: kentroller/cluster-controller pods show `CreateContainerConfigError` with
`couldn't find key CLUSTER_ID in ConfigMap castai-agent/castai-agent-metadata`, and
castai-agent logs show:

```
registering cluster: building register cluster request: getting cluster name:
describing instance_id=i-...: operation error EC2: DescribeInstances, ... context canceled
```

Cause: castai-agent needs `ec2:DescribeInstances` to register the cluster. Without
a Pod Identity association, its AWS SDK falls back to the MNG node instance
profile, which doesn't have that permission. This example creates a dedicated
`<cluster_name>-castai-workload` role with a narrow inline policy
(`ec2:Describe*` for inventory, `pricing:GetProducts`, `eks:DescribeCluster`) and
binds it to the `castai-agent` and `castai-cluster-controller` ServiceAccounts
via `aws_eks_pod_identity_association.castai` in `castai.tf` — if you removed
that block (or its role / policy attachments), re-add it.

### `helm_release.castai` fails with `context deadline exceeded`

Symptom: `terraform apply` exits non-zero with
`Error: context deadline exceeded`, even though `kubectl -n castai-agent get pods`
shows pods are still rolling.

Cause: nine subcharts + image pulls + Karpenter-provisioned nodes coming up can
take 5-7 min on first install, and helm's default timeout is 5 min. This example
sets `timeout = 600` on `helm_release.castai`. If you still hit it, bump higher
or re-run `terraform apply` — the second pass typically converges once images
are cached on nodes.

### `OCI Registry Login Failed: ... keychain ... (-25299)` on macOS

Symptom: `terraform apply` fails during the `helm_release.karpenter` step with:

```
Error: OCI Registry Login Failed: The specified item already exists in the keychain. (-25299)
```

Cause: the helm provider v3 caches an expired ECR public auth token in the macOS
keychain. Clear it:

```sh
security delete-internet-password -s public.ecr.aws
```

Then re-run `terraform apply`.

## Variables

| Name                         | Type        | Default               | Description                                                                                                     |
|------------------------------|-------------|-----------------------|-----------------------------------------------------------------------------------------------------------------|
| `cluster_name`               | string      | —                     | EKS cluster name.                                                                                               |
| `cluster_region`             | string      | —                     | AWS region.                                                                                                     |
| `cluster_version`            | string      | `1.35`                | EKS Kubernetes version.                                                                                         |
| `tags`                       | map(string) | `{}`                  | Tags applied to AWS resources.                                                                                  |
| `castai_api_token`           | string      | —                     | **Sensitive.** CAST AI org API token. Stored in `terraform.tfstate` (matches every other example in this repo). |
| `castai_api_url`             | string      | `https://api.cast.ai` | CAST AI API URL.                                                                                                |
| `castai_grpc_url`            | string      | `grpc.cast.ai:443`    | CAST AI gRPC URL — Kentroller's `StreamActions` target.                                                         |

## Destroy

Run `terraform destroy` in one shot. Terraform's dependency graph removes the
`castai` Helm release (which uninstalls Kentroller cleanly) before EKS goes
away.

If you destroy partially or get stuck:

1. Uninstall the `castai` release first:
   ```sh
   helm uninstall castai -n castai-agent
   ```
2. Delete remaining Karpenter resources:
   ```sh
   kubectl delete nodepools --all
   kubectl delete nodeclaims --all
   ```
3. Then `terraform destroy` the rest.

## Enabling / disabling other CAST AI features

The umbrella's `kent` subchart bundles WOOP, CLM, pod-mutator, spot-handler,
kvisor, and others — all enabled by default under `kent.enabled=true`. Override
a component's `enabled` value to disable it:

```hcl
values = [yamlencode({
  kent = {
    enabled            = true
    castai-live        = { enabled = false }   # CLM
    castai-kvisor      = { enabled = false }   # security
    castai-pod-mutator = { enabled = false }   # admission webhook
  }
  global = {
    castai = {
      apiKey   = var.castai_api_token
      apiURL   = var.castai_api_url
      grpcURL  = var.castai_grpc_url
      provider = "eks"
    }
  }
})]
```

CLM ships installed-but-dormant (controller `replicaCount: 0`). To enable live
migrations, scale the controller up:

```hcl
kent = {
  enabled = true
  castai-live = {
    controller = { replicaCount = 2 }
  }
}
```
