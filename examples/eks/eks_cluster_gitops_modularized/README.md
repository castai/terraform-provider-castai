# EKS + CAST AI GitOps onboarding as a reusable module

This is the [`eks_cluster_gitops`](../eks_cluster_gitops) example reshaped into a **reusable module**.
Instead of hardcoding one node configuration and three node templates in `castai.tf`, the whole
node configuration / node template surface is driven by two input maps:

- `node_configurations` — `map(object(...))`, keyed by configuration name
- `node_templates` — `map(object(...))`, keyed by template name

That makes it consumable from any wrapper where each cluster only supplies its own inputs, and the
module code stays shared.

Like the original example, this covers only the **Terraform half** of GitOps onboarding: it creates
the IAM resources, connects the cluster to CAST AI, and configures node configurations, node
templates and the autoscaler. Installing `castai-agent`, `castai-cluster-controller`,
`castai-evictor`, `castai-spot-handler` and `castai-kvisor` is left to your GitOps tool (Argo CD,
Flux, …) using the `cluster_id` and `cluster_token` outputs. See the
[original example's README](../eks_cluster_gitops/README.md) for the Helm chart values and the
onboarding sequence — this module is a drop-in replacement for its Terraform step.

## Usage

Copy [`terraform.tfvars.example`](./terraform.tfvars.example) to `terraform.tfvars`, fill it in, then:

```sh
terraform init
terraform apply
```

Or reference it as a module:

```hcl
module "castai" {
  source = "github.com/castai/terraform-provider-castai//examples/eks/eks_cluster_gitops_modularized?ref=v8.52.0"

  aws_account_id     = "123456789012"
  aws_cluster_region = "us-east-1"
  aws_cluster_name   = module.eks.cluster_name
  vpc_id             = module.vpc.vpc_id
  castai_api_token   = var.castai_api_token

  subnets = module.vpc.private_subnets
  security_group_ids = [
    module.eks.cluster_security_group_id,
    module.eks.node_security_group_id,
  ]

  node_configurations = {
    default = {
      disk_cpu_ratio = 0
      min_disk_size  = 100
    }
  }

  node_templates = {
    default-by-castai = {
      is_default   = true
      should_taint = false
      constraints  = { on_demand = true }
    }
  }
}
```

If your wrapper generates provider blocks, delete `providers.tf` from your vendored copy so the
generated configuration is the only source of provider settings.

## How the two maps work

### `node_configurations`

The map key becomes the node configuration name. Every attribute is optional; anything you omit
falls back to the CAST AI default, with three convenience fallbacks:

| Attribute | Falls back to |
| --- | --- |
| `subnets` | `var.subnets` |
| `eks.security_groups` | `var.security_group_ids` |
| `eks.instance_profile_arn` | instance profile created by the `castai/eks-role-iam/castai` module |

So a minimal single-configuration setup needs nothing but the module-level `subnets` and
`security_group_ids`, and per-configuration overrides are opt-in:

```hcl
node_configurations = {
  default = {
    disk_cpu_ratio = 0
    min_disk_size  = 100
  }

  storage = {
    min_disk_size = 500
    subnets       = ["subnet-aaa", "subnet-bbb"] # overrides var.subnets

    eks = {
      volume_type       = "gp3"
      volume_iops       = 6000
      volume_throughput = 250
    }
  }
}
```

`default_node_configuration` names the key that gets promoted via
`castai_node_configuration_default` (defaults to `"default"`). A precondition fails the plan if that
key is missing.

### `node_templates`

The map key becomes the template name. `node_configuration` references a **key of
`node_configurations`** — not a raw ID — so units never have to thread IDs around; omit it and the
template attaches to `default_node_configuration`.

```hcl
node_templates = {
  spot = {
    node_configuration = "default"
    should_taint       = true

    custom_labels = { type = "spot" }
    custom_taints = [{ key = "dedicated", value = "spot", effect = "NoSchedule" }]

    constraints = {
      spot                  = true
      use_spot_fallbacks    = true
      enable_spot_diversity = true
      architectures         = ["amd64"]

      instance_families = { exclude = ["m5"] }
      custom_priority   = [{ instance_families = ["c5"], spot = true }]
    }
  }
}
```

`constraints` supports the common scalars plus the nested blocks `instance_families`,
`custom_priority`, `dedicated_node_affinity`, `gpu` and `aws.capacity_reservations`. Blocks are only
emitted when you supply them, so leaving them out keeps the CAST AI defaults. See
`variables.tf` for the full object type.

### `autoscaler_settings`

A flattened object with the same defaults as the original example. Set it to `null` to stop managing
`castai_autoscaler` entirely (useful if the autoscaler is owned by another unit).

## Inputs

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| `aws_account_id` | `string` | — | ID of the AWS account the cluster lives in |
| `aws_cluster_region` | `string` | — | Cluster region |
| `aws_cluster_name` | `string` | — | Cluster name |
| `castai_api_token` | `string` | — | CAST AI API token |
| `vpc_id` | `string` | — | EKS cluster VPC ID |
| `subnets` | `list(string)` | — | Default subnets for node configurations |
| `security_group_ids` | `list(string)` | — | Default security groups for node configurations |
| `node_configurations` | `map(object)` | one `default` configuration | Node configurations to create |
| `default_node_configuration` | `string` | `"default"` | Which configuration becomes the cluster default |
| `node_templates` | `map(object)` | one `default-by-castai` template | Node templates to create |
| `autoscaler_settings` | `object` | see `variables.tf` | Autoscaler settings; `null` disables management |
| `castai_api_url` | `string` | `https://api.cast.ai` | CAST AI API URL |
| `delete_nodes_on_disconnect` | `bool` | `false` | Delete CAST AI nodes on cluster destroy |
| `profile` | `string` | `"default"` | AWS CLI profile |

## Outputs

| Name | Description |
| --- | --- |
| `cluster_id` | CAST AI cluster ID (needed by the Helm charts) |
| `cluster_token` | CAST AI cluster token (sensitive; needed by the Helm charts) |
| `node_configuration_ids` | Map of configuration name → CAST AI node configuration ID |
| `node_template_names` | Names of the managed node templates |
| `instance_profile_arn` / `instance_profile_role_arn` / `cast_role_arn` | IAM resources created for CAST AI |
