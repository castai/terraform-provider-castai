# CAST AI Omni Provisioner Terraform Provider Implementation

This document describes the implementation of Terraform resources for the CAST AI Omni Provisioner API.

## Overview

The Omni Provisioner API allows you to manage edge computing infrastructure across multiple cloud providers (AWS, GCP, OCI) through a unified interface. This implementation adds Terraform support for:

- **Clusters**: Omni cluster onboarding and management
- **Edge Locations**: Geographic deployment zones with cloud provider credentials
- **Edge Configurations**: Templates for standardized node deployments
- **Edges**: Individual compute nodes (VMs/instances)

## Implementation Details

### 1. SDK Client Generation

**Location**: `castai/sdk/omni_provisioner/`

- Auto-generated from OpenAPI spec at `https://api.cast.ai/spec/omni/openapi.yaml`
- Client configuration in `castai/sdk/omni_provisioner/client.go`
- Generation configured in `castai/sdk/generate.go`
- Makefile updated to include `OMNI_PROVISIONER_API_TAGS=OmniProvisionerAPI`

**Generated Files**:
- `api.gen.go` - Type definitions
- `client.gen.go` - HTTP client methods
- `mock/client.go` - Mock client for testing

### 2. Provider Configuration

**File**: `castai/provider.go`

Added `omniProvisionerClient` to `ProviderConfig` struct:
```go
type ProviderConfig struct {
    api                          sdk.ClientWithResponsesInterface
    clusterAutoscalerClient      cluster_autoscaler.ClientWithResponsesInterface
    omniProvisionerClient        omni_provisioner.ClientWithResponsesInterface
    organizationManagementClient organization_management.ClientWithResponsesInterface
}
```

Client is initialized in `providerConfigure()` function.

### 3. Resources

#### 3.1 `castai_omni_cluster`

**File**: `castai/resource_omni_cluster.go`

Manages Omni cluster onboarding lifecycle.

**Schema**:
- `name` (required) - Cluster name
- `organization_id` (required, ForceNew) - Organization ID
- `service_account_id` (optional) - Service account for authentication
- `provider_type` (computed) - Cloud provider type (GKE, EKS)
- `state` (computed) - Current cluster state
- `status` (computed) - Current cluster status
- `onboarding_script` (computed) - Script to onboard the cluster

**API Mapping**:
- Create: `POST /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{id}:onboard`
- Read: `GET /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{id}`
- Delete: Not supported (clusters managed externally)

#### 3.2 `castai_omni_edge_location`

**File**: `castai/resource_omni_edge_location.go`

Manages edge locations (geographic deployment zones) with cloud provider configuration.

**Schema**:
- `organization_id` (required, ForceNew)
- `cluster_id` (required, ForceNew)
- `name` (required)
- `region` (required)
- `zones` (optional)
- `description` (optional)
- `state` (computed)
- `total_edge_count` (computed)
- `aws` (optional, max 1) - AWS configuration block
- `gcp` (optional, max 1) - GCP configuration block
- `oci` (optional, max 1) - OCI configuration block

**AWS Configuration**:
- `account_id` (required)
- `access_key_id` (required, sensitive)
- `secret_access_key` (required, sensitive)
- `vpc_id` (required)
- `subnet_ids` (required)
- `security_group_id` (optional)

**GCP Configuration**:
- `project_id` (required)
- `service_account_json_base64` (required, sensitive)
- `network_name` (required)
- `subnet_name` (required)
- `tags` (optional)

**OCI Configuration**:
- `tenancy_id` (required)
- `compartment_id` (required)
- `user_id` (required)
- `fingerprint` (required)
- `private_key_base64` (required, sensitive)
- `vcn_id` (required)
- `subnet_id` (required)

**API Mapping**:
- Create: `POST /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations`
- Onboard: `POST /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{id}:onboard`
- Read: `GET /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{id}`
- Update: `PATCH /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{id}`
- Delete:
  1. `POST /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{id}:offboard`
  2. `DELETE /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{id}`

#### 3.3 `castai_omni_edge`

**File**: `castai/resource_omni_edge.go`

Manages individual compute nodes (edges).

**Schema**:
- `organization_id` (required, ForceNew)
- `cluster_id` (required, ForceNew)
- `edge_location_id` (required, ForceNew)
- `name` (optional, ForceNew)
- `instance_type` (required, ForceNew) - e.g., "m5.xlarge", "n1-standard-4"
- `scheduling_type` (optional, ForceNew) - "ON_DEMAND" or "SPOT", default: "ON_DEMAND"
- `zone` (optional, ForceNew)
- `node_architecture` (optional, ForceNew) - "X86_64" or "ARM64", default: "X86_64"
- `boot_disk_gib` (optional, ForceNew)
- `image_id` (optional, ForceNew)
- `instance_labels` (optional, ForceNew)
- `kubernetes_labels` (optional, ForceNew)
- `kubernetes_taints` (optional, ForceNew)
- `gpu_config` (optional, ForceNew)
- `configuration_id` (optional, ForceNew)
- `phase` (computed)
- `provider_id` (computed)
- `kubernetes_name` (computed)

**GPU Configuration**:
- `count` (required) - Number of GPUs
- `type` (optional) - GPU type
- `mig` (optional) - MIG configuration
  - `memory_gb` (optional)
  - `partition_sizes` (optional)
- `time_sharing` (optional) - Time-sharing configuration
  - `replicas` (required)

**Kubernetes Taints**:
- `key` (required)
- `value` (optional)
- `effect` (required) - "NoSchedule", "PreferNoSchedule", or "NoExecute"

**API Mapping**:
- Create: `POST /omni-provisioner/v2beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{edgeLocationId}/edges`
- Read: `GET /omni-provisioner/v2beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{edgeLocationId}/edges/{id}`
- Update: Not supported (edges are immutable)
- Delete: `DELETE /omni-provisioner/v2beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{edgeLocationId}/edges/{id}`

### 4. Data Sources

#### 4.1 `castai_omni_edge_configuration`

**File**: `castai/data_source_omni_edge_configuration.go`

Retrieves edge configuration templates.

**Schema**:
- `organization_id` (required)
- `cluster_id` (required)
- `edge_location_id` (optional) - Filter by edge location
- `name` (optional) - Filter by configuration name
- `version` (computed)
- `default` (computed)
- `edge_count` (computed)
- `aws` (computed) - AWS configuration details
- `gcp` (computed) - GCP configuration details

**API Mapping**:
- Read: `GET /omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-configurations`

## Usage Examples

See `examples/omni/main.tf` for comprehensive examples including:

1. **Cluster Onboarding**
   ```hcl
   resource "castai_omni_cluster" "example" {
     name               = "my-omni-cluster"
     organization_id    = var.organization_id
     service_account_id = "service-account-uuid"
   }
   ```

2. **AWS Edge Location**
   ```hcl
   resource "castai_omni_edge_location" "aws_us_east" {
     organization_id = var.organization_id
     cluster_id      = var.cluster_id
     name            = "aws-us-east-1"
     region          = "us-east-1"
     zones           = ["us-east-1a", "us-east-1b"]

     aws {
       account_id        = "123456789012"
       access_key_id     = var.aws_access_key_id
       secret_access_key = var.aws_secret_access_key
       vpc_id            = "vpc-12345678"
       subnet_ids        = ["subnet-1234", "subnet-5678"]
     }
   }
   ```

3. **GCP Edge Location**
   ```hcl
   resource "castai_omni_edge_location" "gcp_us_central" {
     organization_id = var.organization_id
     cluster_id      = var.cluster_id
     name            = "gcp-us-central1"
     region          = "us-central1"

     gcp {
       project_id                   = "my-gcp-project"
       service_account_json_base64  = var.gcp_sa_json
       network_name                 = "default"
       subnet_name                  = "default"
     }
   }
   ```

4. **On-Demand Compute Node**
   ```hcl
   resource "castai_omni_edge" "worker" {
     organization_id  = var.organization_id
     cluster_id       = var.cluster_id
     edge_location_id = castai_omni_edge_location.aws_us_east.id

     name              = "worker-1"
     instance_type     = "m5.xlarge"
     scheduling_type   = "ON_DEMAND"
     zone              = "us-east-1a"
     boot_disk_gib     = 100

     kubernetes_labels = {
       "environment" = "production"
     }
   }
   ```

5. **Spot Instance**
   ```hcl
   resource "castai_omni_edge" "spot_worker" {
     organization_id  = var.organization_id
     cluster_id       = var.cluster_id
     edge_location_id = castai_omni_edge_location.aws_us_east.id

     name            = "spot-worker-1"
     instance_type   = "m5.2xlarge"
     scheduling_type = "SPOT"
     zone            = "us-east-1b"
   }
   ```

6. **GPU Worker with Time-Sharing**
   ```hcl
   resource "castai_omni_edge" "gpu_worker" {
     organization_id  = var.organization_id
     cluster_id       = var.cluster_id
     edge_location_id = castai_omni_edge_location.aws_us_east.id

     name          = "gpu-worker-1"
     instance_type = "p3.2xlarge"

     gpu_config {
       count = 1
       type  = "nvidia-tesla-v100"

       time_sharing {
         replicas = 4
       }
     }
   }
   ```

## Building and Testing

### Generate SDK

```bash
make generate-sdk
```

This will:
1. Download the OpenAPI spec from `https://api.cast.ai/spec/omni/openapi.yaml`
2. Generate Go types and client code
3. Generate mock clients for testing

### Build Provider

```bash
make build
```

This will:
1. Generate SDK code
2. Generate documentation
3. Build the provider binary

### Run Tests

```bash
# Unit tests
make test

# Acceptance tests (requires API credentials)
TF_ACC=1 go test ./castai/... -run='^TestAccOmni_' -v -timeout 50m
```

## File Structure

```
castai/
├── sdk/
│   └── omni_provisioner/
│       ├── api.gen.go           # Generated types
│       ├── client.gen.go        # Generated client
│       ├── client.go            # Client configuration
│       └── mock/
│           └── client.go        # Mock client
├── provider.go                  # Provider registration
├── resource_omni_cluster.go     # Cluster resource
├── resource_omni_edge_location.go # Edge location resource
├── resource_omni_edge.go        # Edge resource
└── data_source_omni_edge_configuration.go # Edge config data source

examples/
└── omni/
    └── main.tf                  # Usage examples

Makefile                         # Updated with OMNI_PROVISIONER_* variables
```

## API Tag Configuration

The Omni Provisioner API tag is configured in the Makefile:

```makefile
export OMNI_PROVISIONER_API_TAGS ?= OmniProvisionerAPI
export OMNI_PROVISIONER_SWAGGER_LOCATION ?= https://api.cast.ai/spec/omni/openapi.yaml
```

## Security Considerations

1. **Sensitive Data**: Cloud credentials (AWS access keys, GCP service account JSON, OCI private keys) are marked as `sensitive` in the schema
2. **State Storage**: Use encrypted remote state backends for production
3. **Credential Management**: Consider using external secret management systems (HashiCorp Vault, AWS Secrets Manager, etc.)

## Limitations and Notes

1. **Cluster Deletion**: The Omni API doesn't support cluster deletion via API. Clusters must be removed manually through the console or API directly.
2. **Edge Immutability**: Once created, edges cannot be updated. Changes require recreation (ForceNew).
3. **Edge Location Updates**: Only `name` and `description` can be updated. Cloud provider configurations are immutable.
4. **API Versions**: The implementation uses both v1beta and v2beta endpoints as defined in the OpenAPI spec.

## Future Enhancements

1. **Additional Resources**:
   - Operations tracking resource
   - Cloud init script customization
   - Peering configuration

2. **Testing**:
   - Unit tests for all resources
   - Acceptance tests
   - Integration tests

3. **Documentation**:
   - Auto-generated Terraform documentation using `tfplugindocs`
   - More comprehensive examples
   - Migration guides

## API Reference

Full API documentation: https://api.cast.ai/spec/omni/openapi.yaml

Key endpoints:
- Clusters: `/omni-provisioner/v1beta/organizations/{organizationId}/clusters`
- Edge Locations: `/omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations`
- Edges (v2): `/omni-provisioner/v2beta/organizations/{organizationId}/clusters/{clusterId}/edge-locations/{edgeLocationId}/edges`
- Configurations: `/omni-provisioner/v1beta/organizations/{organizationId}/clusters/{clusterId}/edge-configurations`
