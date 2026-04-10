## Existing EKS cluster with CAST AI and AI Optimizer

This example shows how to onboard an existing EKS cluster to CAST AI with AI Optimizer enabled for LLM model serving. It includes:

- CAST AI connection with autoscaler and cost optimization
- AI Optimizer Helm chart installation with a GPU node template for LLM workloads
- Optional NVIDIA device plugin Helm chart for GPU node support
- Optional predefined HuggingFace model deployment (Llama 3.2 1B Instruct)
- Optional custom private model deployment from S3

IAM policies required to connect the cluster to CAST AI are created by [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

Example configuration should be analysed in the following order:
1. Creates IAM and other CAST AI related resources to connect EKS cluster to CAST AI, configure Autoscaler and Node Configurations — `castai.tf`
2. Optionally installs NVIDIA device plugin — `nvidia_device_plugin.tf`
3. Optionally deploys predefined model — `ai_optimizer_hosted_model.tf`
4. Optionally deploys custom model from S3 — `ai_optimizer_hosted_model_custom.tf`

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` with your cluster name, cluster region and CAST AI API token.

| Variable | Description |
| --- | --- |
| cluster_name | Name of existing EKS cluster |
| cluster_region | AWS region where the cluster is located |
| castai_api_token | CAST AI API token |
| castai_organization_id | CAST AI organization ID (required when token has access to multiple organizations) |
| delete_nodes_on_disconnect | Delete CAST AI provisioned nodes on cluster disconnect (default: true) |
| enable_ai_optimizer | Enable AI Optimizer (default: true) |
| deploy_predefined_model | Deploy predefined Llama model (default: false) |
| hf_token | HuggingFace token for accessing gated models (required when deploy_predefined_model is true) |
| deploy_custom_model | Deploy custom model from S3 (default: false) |
| custom_model_registry_bucket | S3 bucket name for private model registry |
| custom_model_registry_region | AWS region for the model registry S3 bucket (default: us-east-1) |
| custom_model_name | Name for the custom model (default: my-custom-model) |
| custom_base_model_spec_id | UUID of the base model specs for the custom private registry model |
| install_nvidia_device_plugin | Install NVIDIA device plugin for GPU support (default: false) |
| tags | Optional tags for new cluster nodes |

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
> If your EKS cluster authentication mode is API/API_CONFIGMAP — Terraform will update the role in IAM access entries (no user action needed).
> If your EKS cluster authentication mode is CONFIGMAP — you need to also update the [aws-auth](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) configmap. The instance profile used by CAST AI must be present in the configmap.

## Accessing Deployed Models

Once deployed, models are accessible via Kubernetes services.

Predefined Llama 3.2 1B model (service name: `llama3-2-1b`, port: `11434`):

```bash
kubectl port-forward -n castai-agent service/llama3-2-1b 8080:11434

curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Custom model (service name: `<custom_model_name>-service`, port: `8080`):

```bash
kubectl port-forward -n castai-agent service/<custom_model_name>-service 8080:8080

curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "<custom_model_name>",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
