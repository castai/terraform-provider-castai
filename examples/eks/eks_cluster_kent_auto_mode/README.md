# EKS Auto Mode + Kentroller

Creates an EKS cluster with Auto Mode enabled and installs the CAST AI umbrella chart with `kent.enabled=true`.

## Differences from `eks_cluster_kent`

| | `eks_cluster_kent` | `eks_cluster_kent_auto_mode` |
|---|---|---|
| Karpenter | Stand Helm install + IAM + SQS | Managed by EKS Auto Mode (no install needed) |
| NodeClass CRD | `karpenter.k8s.aws/v1/EC2NodeClass` | `eks.amazonaws.com/v1/NodeClass` (created by Auto Mode) |
| Managed node group | m5.large × 2 (runs Karpenter controller) | None — Auto Mode provisions all nodes |
| IMDS hop limit | Set to 2 via `EC2NodeClass.spec.metadataOptions` | NodeClass CRD has no such field; agent uses `EKS_REGION`/`EKS_CLUSTER_NAME`/`EKS_ACCOUNT_ID` env vars to bypass IMDS |
| Preflight check | Enabled (verifies Karpenter Deployment exists) | Disabled (no standalone Karpenter Deployment on Auto Mode) |

## Prerequisites

- `terraform` >= 1.3.2
- AWS credentials with permissions to create EKS clusters, VPCs, IAM roles, and Pod Identity associations
- CAST AI API token (console.cast.ai → Settings → API access)

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: set cluster_name, cluster_region, castai_api_token

terraform init
terraform apply
```

## Verify

```bash
aws eks update-kubeconfig --name <cluster_name> --region <region>
kubectl get nodes
kubectl get nodepools.karpenter.sh
kubectl get nodeclass default
kubectl get pods -n castai-agent
```
