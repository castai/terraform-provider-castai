## Existing EKS cluster with CAST AI and AI Optimizer

This example shows how to onboard an existing EKS cluster to CAST AI with AI Optimizer enabled for LLM model serving. It includes:

- CAST AI connection with autoscaler and cost optimization
- GPU node template for LLM workloads
- AI Optimizer Helm chart installation
- Optional predefined HuggingFace model deployment (Llama 3.2 1B Instruct)
- Optional custom private model deployment from S3

IAM policies required to connect the cluster to CAST AI are created by [castai/eks-role-iam/castai module](https://github.com/castai/terraform-castai-eks-role-iam).

Example configuration should be analysed in the following order:
1. Creates IAM and other CAST AI related resources to connect EKS cluster to CAST AI, configure Autoscaler and Node Configurations — `castai.tf`
2. Optionally deploys predefined model — `ai_optimizer_hosted_model.tf`
3. Optionally deploys custom model from S3 — `ai_optimizer_hosted_model_custom.tf`

# Usage
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` with your cluster name, cluster region and CAST AI API token.

| Variable | Description |
| --- | --- |
| cluster_name | Name of existing EKS cluster |
| cluster_region | AWS region where the cluster is located |
| castai_api_token | CAST AI API token |
| castai_organization_id | CAST AI organization ID (required when token has access to multiple organizations) |
| enable_ai_optimizer | Enable AI Optimizer (default: true) |
| deploy_predefined_model | Deploy predefined Llama model (default: false) |
| hf_token | HuggingFace token for accessing gated models (required when deploy_predefined_model is true) |
| deploy_custom_model | Deploy custom model from S3 (default: false) |
| custom_model_registry_bucket | S3 bucket for custom model registry |

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

Once deployed, models are accessible via Kubernetes services:

```bash
kubectl port-forward -n castai-agent service/llama3-2-1b 8080:11434

curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
