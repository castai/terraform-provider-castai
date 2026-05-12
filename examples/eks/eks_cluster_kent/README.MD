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

| Check | Command |
|---|---|
| Karpenter running | `kubectl -n karpenter get pods` |
| Karpenter CRs applied | `kubectl get nodepools.karpenter.sh,ec2nodeclasses.karpenter.k8s.aws` |
| Kentroller running | `kubectl -n castai-agent get pods \| grep kentroller` |
| Stream connected | `kubectl -n castai-agent logs deploy/castai-kentroller \| grep -i 'stream'` |
| Cluster visible in CAST AI | https://console.cast.ai → your cluster, Kent indicator on |

## Variables

| Name | Type | Default | Description |
|---|---|---|---|
| `cluster_name` | string | — | EKS cluster name. |
| `cluster_region` | string | — | AWS region. |
| `cluster_version` | string | `1.34` | EKS Kubernetes version. |
| `tags` | map(string) | `{}` | Tags applied to AWS resources. |
| `castai_api_token` | string | — | **Sensitive.** CAST AI org API token. Stored in `terraform.tfstate` (matches every other example in this repo). |
| `castai_api_url` | string | `https://api.cast.ai` | CAST AI API URL. |
| `castai_grpc_url` | string | `grpc.cast.ai:443` | CAST AI gRPC URL — Kentroller's `StreamActions` target. |
| `delete_nodes_on_disconnect` | bool | `true` | If true, CAST AI-provisioned nodes are deleted on disconnect. Recommended `false` for production. |

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
