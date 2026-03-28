## Existing EKS cluster and CAST AI example with CAST AI Autoscaler policies and additional Node Configurations

Following example shows how to onboard existing EKS cluster to CAST AI, configure [Autoscaler policies](https://docs.cast.ai/reference/policiesapi_upsertclusterpolicies) and additional [Node Configurations](https://docs.cast.ai/docs/node-configuration/).

IAM policies required to connect the cluster to CAST AI in the example are created by [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

Example configuration should be analysed in the following order:
1. Creates IAM and other CAST AI related resources to connect EKS cluster to CAST AI, configure Autoscaler and Node Configurations  - `castai.tf`

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your cluster name, cluster region, and CAST AI API token.

| Variable | Description |
| --- | --- |
| `cluster_name` | Name of the EKS cluster |
| `cluster_region` | AWS region where the cluster runs |
| `castai_api_token` | CAST AI API token from console.cast.ai → API Access keys |
| `profile` | AWS CLI profile to use (default: `"default"`) |
| `delete_nodes_on_disconnect` | Delete CAST AI-provisioned nodes on disconnect (default: `true`; set to `false` for production) |
| `tags` | Optional map of tags applied to new nodes |
| `agent_values` | Optional list of YAML Helm values for castai-agent. Example: `[<<-EOF\n  additionalEnv:\n    MY_VAR: "value"\nEOF\n]` — see [Customizing Helm values](#customizing-helm-values-additionalenv-and-more) |
| `cluster_controller_values` | Optional list of YAML Helm values for castai-cluster-controller, same format as `agent_values` |

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
> If you are onboarding existing eks cluster to CAST AI 
0. Set Profile in AWS CLI
AWS CLI profile is already set to default, override if only required.
1. If your eks cluster authentication mode is API/API_CONFIGMAP - TF will update role in IAM access entries (No user action needed)
2. If your eks cluster authentication mode is CONFIGMAP - you need to also update [aws-auth](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) configmap. In the configmap instance profile
> used by CAST AI has to be present. Example of entry can be found [here](https://github.com/castai/terraform-provider-castai/blob/157babd57b0977f499eb162e9bee27bee51d292a/examples/eks/eks_cluster_autoscaler_polices/eks.tf#L28-L38).


# Customizing Helm values (additionalEnv and more)

The `agent_values` and `cluster_controller_values` variables accept lists of YAML-formatted Helm values that are merged into the respective CAST AI Helm chart releases. This is the recommended way to configure environment variables, proxy settings, resource limits, or any other chart-level option.

Example `terraform.tfvars` entry to set environment variables on the agent:

```hcl
agent_values = [<<-EOF
  additionalEnv:
    HTTP_PROXY: "http://proxy.example.com:8080"
    HTTPS_PROXY: "http://proxy.example.com:8080"
    NO_PROXY: "169.254.169.254,localhost"
EOF
]
```

Use `cluster_controller_values` for the same pattern applied to the cluster-controller component.

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
