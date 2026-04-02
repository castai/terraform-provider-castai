# AI Optimizer Complete Example

This example demonstrates deploying an EKS cluster connected to CAST AI with AI Optimizer enabled for LLM model serving. It includes:

- EKS cluster with GPU node groups (g5.xlarge instances)
- CAST AI connection with autoscaler and cost optimization
- AI Optimizer Helm chart installation
- Model registry for custom S3-hosted models
- Predefined HuggingFace model (Llama 3.1 8B Instruct)
- Optional custom private model deployment

## Prerequisites

1. AWS CLI configured with appropriate credentials
2. Terraform >= 1.0
3. CAST AI API token from [console.cast.ai](https://console.cast.ai)
4. (Optional) HuggingFace token for accessing gated models, stored as a Kubernetes secret

## Usage

1. Copy the example variables file and customize:

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
```

2. Initialize Terraform:

```bash
terraform init
```

3. Plan and apply:

```bash
terraform plan
terraform apply
```

## Configuration

### Deploying Only the Predefined Model (Default)

By default, the example deploys Llama 3.1 8B Instruct from HuggingFace:

```hcl
deploy_predefined_model = true
deploy_custom_model     = false
```

### Deploying a Custom Model

To deploy a custom model from S3:

```hcl
deploy_predefined_model = false
deploy_custom_model     = true
model_registry_bucket   = "my-company-model-bucket"
custom_model_name       = "my-fine-tuned-model"
```

### Deploying Both Models

```hcl
deploy_predefined_model = true
deploy_custom_model     = true
model_registry_bucket   = "my-company-model-bucket"
custom_model_name       = "my-custom-model"
```

## HuggingFace Token Secret

For gated models like Llama, create the token secret in your cluster:

```bash
kubectl create namespace castai-agent --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic huggingface-token \
  --from-literal=token="hf_..." \
  -n castai-agent
```

## Accessing Deployed Models

Once deployed, models are accessible via Kubernetes services:

```bash
# Port-forward to the Llama 3.1 service
kubectl port-forward -n castai-agent service/llama31-service 8080:8080

# Test the model
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-3.1-8b-instruct",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Clean Up

```bash
terraform destroy
```

## Resources Created

| Resource | Description |
|----------|-------------|
| `castai_eks_clusterid` | CAST AI cluster connection |
| `castai_ai_optimizer_model_registry` | S3-based model registry (optional) |
| `castai_ai_optimizer_model_specs` | Model specifications for both predefined and custom models |
| `castai_ai_optimizer_hosted_model` | Deployed model instances with autoscaling |
| `helm_release.ai_optimizer` | AI Optimizer Helm chart installation |
| `module.eks` | EKS cluster with GPU node groups |
| `module.castai-eks-cluster` | CAST AI cluster configuration |
