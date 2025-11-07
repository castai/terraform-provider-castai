# CAST AI Castware Operator

Deploy the CAST AI Castware Operator to manage CAST AI components declaratively using Terraform.

## Overview

The [Castware Operator](https://docs.cast.ai/docs/castai-operator) manages the lifecycle of CAST AI components in your Kubernetes cluster. This example shows how to deploy the operator and configure components using Terraform.

## Prerequisites
- CAST AI account and [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key)
- Terraform >= 1.0

## Operator Modes

The operator supports three primary modes but we will refer to only 2 of them as the `read` mode is not managing components only discover them:

#### Write Mode (Default)
Manual version control - you specify component versions explicitly.

**Use when:**
- You want to pin specific component versions
- You need approval before upgrades
- Testing new versions before rollout

```hcl
  set {
    name  = "defaultCluster.migrationMode"
    value = "write"
  }
```

#### AutoUpgrade Mode
Automatic version management - the operator handles upgrades.

**Use when:**
- You want automatic updates to latest versions when operator takes over
- You trust CAST AI's release process
- Running production workloads with auto-updates enabled

```hcl
  set {
    name  = "defaultCluster.migrationMode"
    value = "autoUpgrade"
  }
```

**Important:** In `autoUpgrade` mode, do NOT specify explicit component versions in your values file.

## Usage

### 1. Configure Variables

Copy the example file and configure your values:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
cluster_provider = "aks"  # aks, gke, or eks
castai_api_token = "your-castai-api-token"
operator_mode    = "write"  # or "autoUpgrade"
```

### 2. Customize Component Configuration (Optional)

Edit `castware-values.yaml` to override component settings:

```yaml
components:
  castai-agent:
    overrides:
      additionalEnv:
        LOG_LEVEL: "info"
        AKS_CLUSTER_NAME: ${aks_cluster_name}
```

**Available components:**
- `castai-agent` - Main CAST AI agent

See [full configuration options](https://github.com/castai/helm-charts/tree/main/charts/castai-agent) in the agent chart.

### 3. Deploy

```bash
terraform init
terraform plan
terraform apply
```

### 4. Verify

```bash
# Check operator
kubectl get deployment castware-operator -n castai-agent

# Check components
kubectl get components -n castai-agent

# Check pods
kubectl get pods -n castai-agent
```

## Integration with CAST AI Cluster Modules

### When Using CAST AI Cluster Modules

If you're using a CAST AI cluster module (e.g., `castai/aks/castai`, `castai/gke/castai`, `castai/eks/castai`), add the operator **after** the cluster module:

```hcl
# First: CAST AI cluster module
module "castai-aks-cluster" {
  source  = "castai/aks/castai"
  version = "~> 4.0"
  
  # ... cluster configuration
}

# Then: Castware Operator with dependency
resource "helm_release" "castware_operator" {
  name       = "castware-operator"
  namespace  = "castai-agent"
  repository = "https://castai.github.io/helm-charts"
  chart      = "castware-operator"
  # ... operator configuration
  
  depends_on = [
    module.castai-aks-cluster  # Wait for cluster module
  ]
}

# Finally: Components
resource "helm_release" "castware_components" {
  # ... component configuration
  
  depends_on = [
    helm_release.castware_operator
  ]
}
```

### When NOT Using CAST AI Cluster Modules

If you're deploying the operator standalone (no cluster module), you don't need the `depends_on` for the module:

```hcl
# Just operator
resource "helm_release" "castware_operator" {
  name       = "castware-operator"
  namespace  = "castai-agent"
  repository = "https://castai.github.io/helm-charts"
  chart      = "castware-operator"
  # ... operator configuration
}

# Then components
resource "helm_release" "castware_components" {
  # ... component configuration
  
  depends_on = [
    helm_release.castware_operator
  ]
}
```

## Overriding Component Values

There are two ways to override component configurations:

### Method 1: Using Values File

Edit `castware-values.yaml`:

```yaml
components:
  castai-agent:
    component: castai-agent
    cluster: castai
    enabled: true
    overrides:
      replicaCount: 3
      additionalEnv:
        AKS_CLUSTER_NAME: ${aks_cluster_name}
        AKS_CLUSTER_REGION: ${aks_cluster_region}
```

```hcl
resource "helm_release" "castware_components" {
  # ... other config
  
  values = [
    templatefile("${path.module}/castware-values.yaml", {
      aks_cluster_name = var.aks_cluster_name
      aks_cluster_region = var.aks_cluster_region
    })
  ]
}
```

### Method 2: Inline Overrides

Add to `main.tf`:

```hcl
resource "helm_release" "castware_components" {
  # ... other config
  
  set {
    name  = "components.castai-agent.overrides.additionalEnv.LOG_LEVEL"
    value = "debug"
  }
  
  set {
    name  = "components.castai-agent.overrides.replicas"
    value = "4"
  }
}
```

### Method 3: Combined Approach

Use the values file for base configuration and inline overrides for environment-specific settings:

```hcl
resource "helm_release" "castware_components" {
  # ... other config
  
  # Base configuration
  values = [file("${path.module}/castware-values.yaml")]
  
  # Environment-specific overrides
  set {
    name  = "components.castai-agent.overrides.additionalEnv.ENVIRONMENT"
    value = var.environment
  }
}
```

## Mode-Specific Configuration

### Write Mode Configuration

Specify component versions in `castware-values.yaml`:

```yaml
components:
  castai-agent:
    version: "v1.2.3"  # Explicit version
    overrides:
      # ... your overrides
```

### AutoUpgrade Mode Configuration

Do NOT specify versions - let the operator manage them:

```yaml
components:
  castai-agent:
    # No version specified
    overrides:
      # ... your overrides
```

## Cleanup

```bash
terraform destroy
```

## Documentation

- [Castware Operator Documentation](https://docs.cast.ai/docs/castai-operator)
- [Troubleshooting Guide](https://docs.cast.ai/docs/castai-operator#troubleshooting)

## Support

For issues or questions:
- [CAST AI Documentation](https://docs.cast.ai/)
- [CAST AI Support Portal](https://support.cast.ai/)
- [Community Forum](https://community.cast.ai/)
