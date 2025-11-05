# AI Optimizer API Integration

This document describes the implementation of AI Optimizer API support in the CAST AI Terraform Provider.

## Overview

The AI Optimizer API provides endpoints for managing AI workloads, including:
- API Keys for authentication
- Analytics and reporting
- API key settings management
- Hosted models lifecycle management
- Component monitoring

## Implementation Details

### 1. SDK Generation

The AI Optimizer client is automatically generated from the OpenAPI specification using `oapi-codegen`.

**Files Modified:**
- `Makefile` - Added AI_OPTIMIZER_API_TAGS and AI_OPTIMIZER_SWAGGER_LOCATION environment variables
- `castai/sdk/generate.go` - Added code generation directives for AI Optimizer API

**Configuration:**
```makefile
export AI_OPTIMIZER_API_TAGS ?= APIKeysAPI,AnalyticsAPI,SettingsAPI,HostedModelsAPI,ComponentsAPI
export AI_OPTIMIZER_SWAGGER_LOCATION ?= https://api.cast.ai/spec/ai-optimizer/openapi.yaml
```

**Generated Files:**
- `castai/sdk/ai_optimizer/api.gen.go` - Type definitions
- `castai/sdk/ai_optimizer/client.gen.go` - API client methods
- `castai/sdk/ai_optimizer/client.go` - Client factory function
- `castai/sdk/ai_optimizer/mock/client.go` - Mock client for testing

### 2. Provider Integration

The AI Optimizer client is integrated into the Terraform provider's configuration.

**Files Modified:**
- `castai/provider.go` - Added aiOptimizerClient to ProviderConfig and instantiation in providerConfigure

**Changes:**
```go
type ProviderConfig struct {
    api                          sdk.ClientWithResponsesInterface
    clusterAutoscalerClient      cluster_autoscaler.ClientWithResponsesInterface
    organizationManagementClient organization_management.ClientWithResponsesInterface
    aiOptimizerClient            ai_optimizer.ClientWithResponsesInterface  // NEW
}
```

### 3. Resources and Data Sources

#### Resource: castai_ai_optimizer_api_key

Creates and manages AI Optimizer API keys for authentication.

**File:** `castai/resource_ai_optimizer_api_key.go`

**Schema:**
- `organization_id` (Required, ForceNew) - CAST AI organization ID
- `name` (Required, ForceNew) - Name of the API key
- `token` (Computed, Sensitive) - The generated API key token

**Example Usage:**
```hcl
resource "castai_ai_optimizer_api_key" "example" {
  organization_id = "your-org-id"
  name            = "my-ai-optimizer-key"
}

output "api_key_token" {
  value     = castai_ai_optimizer_api_key.example.token
  sensitive = true
}
```

**Limitations:**
- The API does not provide endpoints to list or read existing API keys after creation
- API keys cannot be deleted via the API (must be done through the CAST AI console)
- The token is only available at creation time

#### Data Source: castai_ai_optimizer_hosted_models

Retrieves information about hosted AI models in a cluster.

**File:** `castai/data_source_ai_optimizer_hosted_models.go`

**Schema:**
- `organization_id` (Required) - CAST AI organization ID
- `cluster_id` (Required) - CAST AI cluster ID
- `models` (Computed) - List of hosted models with:
  - `id` - Model ID
  - `name` - Model name
  - `status` - Model status (RUNNING, DEPLOYING, FAILED, etc.)

**Example Usage:**
```hcl
data "castai_ai_optimizer_hosted_models" "example" {
  organization_id = "your-org-id"
  cluster_id      = "your-cluster-id"
}

output "models" {
  value = data.castai_ai_optimizer_hosted_models.example.models
}
```

### 4. Documentation

The Terraform provider documentation is automatically generated during the build process:
- `docs/resources/ai_optimizer_api_key.md` - Resource documentation
- `docs/data-sources/ai_optimizer_hosted_models.md` - Data source documentation

## Available API Endpoints

The following AI Optimizer API endpoints are available through the generated SDK:

### API Keys
- `APIKeysAPICreateAPIKey` - Create a new API key
- `APIKeysAPIVerifyAPIKey` - Verify API key validity

### Analytics
- `AnalyticsAPIGenerateAnalytics` - Generate analytics report

### Settings
- `SettingsAPIListAPIKeySettings` - List API key settings
- `SettingsAPIGetAPIKeySettings` - Get specific API key settings
- `SettingsAPIUpsertAPIKeySettings` - Create or update API key settings
- `SettingsAPIDeleteAPIKeySettings` - Delete API key settings
- `SettingsAPIGetSettings` - Get organization settings
- `SettingsAPIUpdateSettings` - Update organization settings
- `SettingsAPIResolveSettings` - Resolve effective settings

### Hosted Models
- `HostedModelsAPIListHostedModels` - List hosted models
- `HostedModelsAPICreateHostedModel` - Create a new hosted model
- `HostedModelsAPIUpdateHostedModel` - Update hosted model configuration
- `HostedModelsAPIDeleteHostedModel` - Delete a hosted model
- `HostedModelsAPIScaleHostedModel` - Scale a hosted model
- `HostedModelsAPIGetHostedModelPods` - Get pod status and events

### Components
- `ComponentsAPIListComponents` - List CASTware components

## Future Enhancements

Additional resources and data sources can be implemented for:

1. **AI Optimizer Settings Management**
   - Resource for API key settings (rate limits, fallback models, etc.)
   - Resource for organization-level settings

2. **Hosted Model Management**
   - Resource for creating and managing hosted models
   - Resource for model scaling configuration
   - Data source for model events and metrics

3. **Analytics and Monitoring**
   - Data source for analytics reports
   - Data source for component status

4. **Advanced Features**
   - Settings resolution and inheritance
   - Hibernation configuration
   - Horizontal autoscaling settings

## Testing

To test the implementation:

```bash
# Generate SDK and build provider
make build

# Run tests (when available)
make test

# Example Terraform configuration
cd examples/ai_optimizer
terraform init
terraform plan
terraform apply
```

## API Reference

- OpenAPI Specification: https://api.cast.ai/spec/ai-optimizer/openapi.yaml
- CAST AI API Documentation: https://api.cast.ai/docs

## Notes

- The AI Optimizer API follows the same authentication pattern as other CAST AI APIs
- All API endpoints require a valid CAST AI API token
- Organization ID and Cluster ID are required for most operations
- The SDK is automatically generated and should not be modified manually
- To update the SDK, modify the API tags in the Makefile and run `make generate-sdk`
