# Cast AI Operator

Deploy the Cast AI Operator to manage Cast AI components declaratively using Terraform.

## Overview

The [Cast AI Operator](https://docs.cast.ai/docs/castai-operator) manages the lifecycle of Cast AI components in your Kubernetes cluster. This example shows how to deploy the Operator and configure components using Terraform.

## Prerequisites
- Cast AI account and [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key)
- Terraform >= 1.0

## Operator Migrations Modes

The Operator supports three primary migrations modes but we will refer to only 2 of them as the `read` mode is not managing components only discover them:

#### Write Mode (Default)
Manual version control - you specify component versions explicitly.

**Use when:**
- You want to pin specific component versions
- You need approval before upgrades
- Testing new versions before rollout
- You don't want the agent & spot-handler versions to be updated during Operator onboarding

```hcl
  set {
    name  = "defaultCluster.migrationMode"
    value = "write"
  }
```

#### AutoUpgrade Mode
Automatic version management - the operator handles upgrades to latest version when onboarded.
When Operator is onboarded with this migration mode, it will upgrade the component version to latest, regardless of what is already installed.
This mode will not upgrade the Operator version to latest automatically.

**Use when:**
- You want automatic updates to latest versions when operator takes over

```hcl
  set {
    name  = "defaultCluster.migrationMode"
    value = "autoUpgrade"
  }
```

**Important:** In `autoUpgrade` mode, do NOT specify explicit component versions in your values file.

## Usage

### 1. Configure Variables

Edit `terraform.tfvars`:

```hcl
cluster_provider = "aks"  # aks, gke, or eks
castai_api_token = "your-castai-api-token"
...
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
- `castai-agent` - Main Cast AI agent ([full configuration options](https://github.com/castai/helm-charts/tree/main/charts/castai-agent))
- `spot-handler` - Cast AI spot handler daemon ([full configuration options](https://github.com/castai/helm-charts/tree/main/charts/castai-spot-handler)) *support added with Operator version v0.1.0

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

## Integration with Cast AI Cluster Modules

### When Using Cast AI Cluster Modules

If you're using a Cast AI cluster module (e.g., `castai/aks/castai`, `castai/gke/castai`, `castai/eks/castai`), add the operator **after** the cluster module:

```hcl
# First: Cast AI cluster module
module "castai-aks-cluster" {
  source  = "castai/aks/castai"
  version = "~> 4.0"
  
  # ... cluster configuration
}

# Then: `castware-operator` with dependency
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

### When NOT Using Cast AI Cluster Modules

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

- [Cast AI Operator Documentation](https://docs.cast.ai/docs/castai-operator)
- [Troubleshooting Guide](https://docs.cast.ai/docs/castai-operator#troubleshooting)

## Support

For issues or questions:
- [Cast AI Documentation](https://docs.cast.ai/docs/getting-started#/where-to-get-help)
