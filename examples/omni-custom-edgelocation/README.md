# Omni Custom Edge Location

Creates a CAST AI Omni custom edge location using the `castai_edge_location` resource with the `custom` cloud provider.

Custom edge locations don't provision cloud infrastructure — the edge cluster runs on existing nodes in your environment.

## Usage

```hcl
module "omni_custom_edge_location" {
  source = "./omni-custom-edgelocation"

  castai_api_token   = var.castai_api_token
  organization_id    = var.organization_id
  cluster_id         = var.cluster_id
  edge_location_name = "my-custom-edge"
  description        = "My custom edge location"
  control_plane_mode = "DEDICATED"

  tunneled_cidrs    = ["10.0.0.0/8"]
  cni_overlay       = "OVERLAY_FULL"
  cni_overlay_encap = "OVERLAY_ENCAP_IPIP"
}
```

## Prerequisites

- Terraform >= 1.x
- CAST AI provider >= 8.2.0
- A CAST AI Omni cluster already onboarded
- Valid CAST AI API key

## Inputs

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `castai_api_token` | CAST AI API token | `string` | — |
| `organization_id` | CAST AI Organization ID | `string` | — |
| `cluster_id` | Omni cluster ID | `string` | — |
| `edge_location_name` | Name of the edge location | `string` | — |
| `description` | Description of the edge location | `string` | `"Custom edge location onboarded by Terraform"` |
| `control_plane_mode` | Control plane mode | `string` | `"DEDICATED"` |
| `tunneled_cidrs` | CIDRs routed through main cluster | `list(string)` | `[]` |
| `cni_overlay` | CNI overlay mode | `string` | `"OVERLAY_FULL"` |
| `cni_overlay_encap` | CNI encapsulation protocol | `string` | `"OVERLAY_ENCAP_IPIP"` |

## CNI Values

| `cni_overlay` | `cni_overlay_encap` |
|---------------|---------------------|
| `OVERLAY_UNSPECIFIED` | `OVERLAY_ENCAP_UNSPECIFIED` |
| `OVERLAY_OFF` | `OVERLAY_ENCAP_IPIP` |
| `OVERLAY_SUBNET` | `OVERLAY_ENCAP_FOU` |
| `OVERLAY_FULL` | |

## Outputs

| Name | Description |
|------|-------------|
| `edge_location_id` | ID of the created edge location |
| `edge_location_name` | Name of the edge location |
| `credentials_revision` | Revision number for credentials |