## Existing EKS cluster with CAST AI Storage Autoscaling and Kvisor storage metrics

Following example shows how to onboard an existing EKS cluster to CAST AI and enable [Storage Autoscaling](https://docs.cast.ai/docs/storage-introduction) together with [Kvisor storage metrics](https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics).

Storage Autoscaling expands PVCs and right-sizes EBS volume IOPS/throughput. Kvisor feeds the Storage autoscaler dashboard in the Cast AI console (savings estimates, storage type breakdown, resize event logs, per-PV usage). Storage Autoscaling still works without Kvisor, but the dashboard stays empty.

Both components call AWS APIs from inside the cluster, so they need dedicated IAM permissions, which are **not** created by the `castai/eks-cluster` module. This example provisions them in `iam.tf` as two separate IRSA roles, so that Kvisor does not inherit the write permissions of the Storage Autoscaler.

**Storage Autoscaler** (`castai-agent/castai-storage-autoscaler` service account), following the [manual EKS IAM setup](https://docs.cast.ai/docs/storage-installation#eks-identity-and-permissions-manual-setup):

| IAM permission | Purpose | Required for |
| --- | --- | --- |
| `ec2:DescribeVolumes` | Read current EBS volume state | capacity + performance |
| `ec2:DescribeVolumesModifications` | Poll in-flight ModifyVolume operations | capacity + performance |
| `ec2:ModifyVolume` | Issue size/IOPS/throughput changes **only** when the volume has at least one of the tags described below | capacity + performance |
| `ec2:DescribeInstanceTypes` | Discover per-VM EBS performance caps | performance only |
| `cloudwatch:GetMetricData` | Fetch actual IOPS/throughput consumption | performance only |

`ec2:ModifyVolume` is restricted by two Allow statements: a volume is eligible if it has **either** `eks:eks-cluster-name` set to `<cluster_name>`, **or** `kubernetes.io/cluster/<cluster_name>` set to `owned`. A volume with neither tag cannot be modified. The two statements implement a logical OR, not a requirement to have both tags. Read-only `ec2:Describe*` and `cloudwatch:GetMetricData` permissions are unchanged.

**Kvisor controller** (`castai-agent/castai-kvisor-controller` service account), following the [Kvisor storage metrics guide](https://docs.cast.ai/docs/sec-configuring-kvisor#kubernetes-storage-metrics):

| IAM permission | Purpose |
| --- | --- |
| `ec2:DescribeVolumes` | Read EBS volume metadata (type, size, provisioned IOPS, throughput) |

Each policy is attached to an IAM role that only the matching Kubernetes service account can assume, using [IAM Roles for Service Accounts (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html). The role ARNs are passed to the `castai/eks-cluster` module via `storage_autoscaler_values` and `kvisor_values`, which annotate the Kubernetes service accounts created by the umbrella Helm chart. `kvisor_values` also turns on storage metrics collection: `storage-stats-enabled` on the Kvisor agent and `cloud-provider-storage-sync-enabled` on the Kvisor controller.

IAM policies required to connect the cluster itself to CAST AI are created by the [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

Example configuration should be analysed in the following order:
1. Creates IAM resources for the Storage Autoscaler and for Kvisor storage metrics - `iam.tf`
2. Creates IAM and other CAST AI related resources to connect the EKS cluster to CAST AI, and enables Storage Autoscaling and Kvisor - `castai.tf`

# Prerequisites
- EKS cluster running Kubernetes 1.29 or later
- The cluster's OIDC provider registered in IAM (EKS registers it automatically for newly created clusters; for older clusters see [Enable IAM Roles for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html))
- A StorageClass with `allowVolumeExpansion: true`

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region and CAST AI API token.

| Variable | Description |
| --- | --- |
| cluster_name                = "" | Name of cluster |
| cluster_region              = "" | Name of region of cluster |
| castai_api_token            = "" | Cast api token |
| profile                     = "" | Optional AWS profile. Leave unset to use `AWS_PROFILE` or the default AWS credential chain; setting it overrides `AWS_PROFILE` |

3. Initialize Terraform. Under example root folder run:
```
terraform init
```
4. Run Terraform apply:
```
terraform apply
```
5. To destroy resources created by this example:
```
terraform destroy
```

> **Note**
>
> - Storage Autoscaling installs in **dry-run mode**: planned actions are logged without modifying PVCs. Review the dry-run logs (`kubectl logs -n castai-agent -l app.kubernetes.io/name=castai-storage-autoscaler --follow`), then enable live mode as described in [Enable live mode](https://docs.cast.ai/docs/storage-installation#enable-live-mode) - e.g. flip `config.dryRun` to `false` in `storage_autoscaler_values` or patch the default PVPolicy.
> - **AWS credentials:** the example no longer forces a profile. `profile` defaults to `null`, so the AWS provider and `aws eks get-token` use the standard credential chain (`AWS_PROFILE`, `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, SSO, then the default profile). Set `profile` only to force one profile. If you see `No valid credential sources found` together with `no EC2 IMDS role found`, the selected profile resolved to no usable credentials, so the SDK fell through to EC2 instance metadata. Check that the selected profile works with `aws sts get-caller-identity` (for SSO profiles run `aws sso login` first).
> - Kvisor storage metrics use IRSA here. Alternatively, Kvisor can authenticate with the node instance role through IMDS when the controller pod can reach it; no IRSA role is needed in that case.
> - The Storage Autoscaler sub-chart is wired under the `autoscaler.` profile. On Karpenter-managed EKS clusters connected with the `kent` profile, address the sub-chart under `kent.` instead (see [Installation](https://docs.cast.ai/docs/storage-installation)).
> - `workload_autoscaler_keep_crds=true` is required with `umbrella_enabled=true`. For a new onboarding, as assumed by this example, no separate migration is needed. When converting an installation that already has standalone CAST AI Helm charts, first apply the standalone configuration with `umbrella_enabled=false` and `workload_autoscaler_keep_crds=true`, using the compatible module version from the [umbrella migration guide](https://github.com/castai/terraform-provider-castai/tree/master/docs/umbrella-migration). Only then enable the umbrella chart in a separate apply.
> - `kvisor_grpc_addr` controls Kvisor's separate `CASTAI_API_GRPC_ADDR` and is independent of `castai_grpc_url`. Use `kvisor.prod-eu.cast.ai:443` for the EU console (host:port only, without `https://`). The main API URL is separately inherited by Kvisor from `global.castai.apiURL`.
> - The `install_storage_autoscaler` and `storage_autoscaler_values` inputs require `castai/eks-cluster` module v17.0.0 or newer ([storage autoscaler support](https://github.com/castai/terraform-castai-eks-cluster/pull/192)).

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
