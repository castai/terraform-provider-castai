# Impersonation for GKE clusters

Cast AI supports GKE service account impersonation as an alternative to using service account keys. This method enhances security by eliminating the need for key management while still allowing Cast AI to access your GCP resources.

For general information about this method, see https://docs.cast.ai/docs/sa-impersonation-gke.

This document describes how to configure GKE Impersonation using Terraform for new clusters and how to migrate existing clusters to Impersonation.

## History

Impersonation was originally supported via the `castai/gke-iam-impersonate` module and the `castai_gke_cluster_id` resource. This is now referred to as "Legacy Impersonation".

In November 2026, a new, simpler way to set up Impersonation was introduced. This new flow (referred to as "New Impersonation") came with the following changes:
- The `castai/gke-iam-impersonate` module was deprecated, it's no longer required to set up Impersonation.
- The `castai/gke-iam` module was enhanced to support Impersonation by specifying `authentication_method=impersonation`. It is now the preferred way to create the GCP resources needed for Impersonation.
- The `castai_gke_cluster_id` resource was deprecated, it's no longer required to set up Impersonation.
- The `castai_gke_cluster` resource was extended with the `target_service_account_email` and `castai_service_account_email` fields. These allow configuring Impersonation, taking the place of the `castai_gke_cluster_id` resource.

The New Impersonation relies on the **same cloud resources** (GCP Service Account and permissions to impersonate it) in your GCP Project, but passes the details of those to Cast AI using the `castai_gke_cluster` resource instead of `castai_gke_cluster_id`.

## What am I using now?

If you have a `google_service_account_key` resource in your Terraform state (called `castai_key` if you used the `castai/gke-iam` module), you're using Service Account Keys (static credentials).

If you have a `castai_gke_cluster_id` resource in your Terraform state (called `cluster_id` if you used the `castai/gke-iam-impersonate` module), you're using "Legacy Impersonation".

If none of the above apply, you're using "New Impersonation". You should see the `target_service_account_email` and `castai_service_account_email` fields set on your `castai_gke_cluster` resource.

## Connecting with Impersonation

If you want to connect a new cluster to Cast AI, the recommended way is "New Impersonation". At the time of writing, the latest [11.1.0 version](https://github.com/castai/terraform-castai-gke-cluster/releases/tag/v11.1.0) of the `castai/gke-cluster` module supports doing that by setting `authentication_method=impersonation`. You can check the [gke_impersonate provider example](https://github.com/castai/terraform-provider-castai/tree/master/examples/gke/gke_impersonate) for a complete example.

## Migrating from Service Account Keys to Impersonation

If your cluster is connected using Service Account Keys (static credentials), you can migrate it to Impersonation by following the process below.

> TODO: similar to "Migrating from Legacy to New Impersonation", describing different permutations of module vs. no-module

## Migrating from Legacy to New Impersonation

If your cluster is connected using Legacy Impersonation, you should migrate it to the New Impersonation. The essence of this is removing the `castai_gke_cluster_id` resource from Terraform state and passing the `target_service_account_email` and `castai_service_account_email` fields to the `castai_gke_cluster` resource instead. GCP resources should stay the same (as much as possible, see details below).

Depending on which Cast AI Terraform modules you relied on, the process is slightly different:

|                          | Used `gke-iam-impersonate`                                                                                                   | Didn't use `gke-iam-impersonate`                                                                                 |
|--------------------------|------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------|
| Used `gke-cluster`       | **A**. `gke-iam-impersonate` module will be replaced with `gke-iam`.<br>The `gke-cluster` module receives different inputs.      | **B**. `castai_gke_cluster_id` resource will be removed.<br>The `gke-cluster` module receives different inputs.      |
| Didn't use `gke-cluster` | **C**. `gke-iam-impersonate` module will be replaced with `gke-iam`.<br>`castai_gke_cluster` resource receives different inputs. | **D**. `castai_gke_cluster_id` resource will be removed.<br>`castai_gke_cluster` resource receives different inputs. |

The following section describes common pre-requisites, and the ones after that show configuration examples for each of these cases.

### Prerequisites

- Terraform **>= 1.7**. the examples below use `removed` blocks, which require it.
- CAST AI provider **>= 9.14.0** or higher:
  ```hcl
  terraform {
    required_providers {
      castai = {
        source  = "castai/castai"
        version = ">= 9.14.0"
      }
    }
  }
  ```
- Keep the same `gke_cluster_name` through the migration: the service account name (`castai-gke-tf-<sha1 of gke_cluster_name>`) and the `castai_role` ID are derived from it. Changing it makes the `moved` blocks miss their targets.

### A) Used `gke-iam-impersonate`, Used `gke-cluster`

The following changes are required. Resources commented out can be deleted from the configuration, others need to be added or changed, see comments.

Changes need to be applied together in a single `terraform apply`.

```terraform
# 1. `castai/gke-iam-impersonate` module needs to be removed
#
# module "iam_impersonate" {
#   source  = "castai/iam-impersonate/gke"
#   # ...
# }

# It needs to be dropped from the state without deleting it
removed {
  from = module.iam_impersonate.castai_gke_cluster_id.cluster_id
  
  lifecycle {
    destroy = false
  }
}

# 2. Add `castai/gke-iam` instead
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name

  authentication_method = "impersonation"
}

# Some resources can be moved to avoid recreating them
moved {
  from = module.iam_impersonate.google_service_account.client_service_account
  to   = module.castai_gke_iam.google_service_account.castai_service_account
}

moved {
  from = module.iam_impersonate.google_project_iam_custom_role.castai_role
  to   = module.castai_gke_iam.google_project_iam_custom_role.castai_role
}

# 3. Change `castai/gke-cluster` inputs
module "castai-gke-cluster" {
  source  = "castai/gke-cluster/castai"
  version = "~> 11.1"

  # Remove gke_credentials:
  # gke_credentials = "{}"
  
  # Add the following instead:
  target_service_account_email = module.castai_gke_iam.target_service_account_email
  castai_service_account_email = module.castai_gke_iam.castai_service_account_email
}
```

The expected changes in the resulting plan are:
- TODO
- TODO
- TODO

### B) Didn't use `gke-iam-impersonate`, Used `gke-cluster`

> TODO

### C) Used `gke-iam-impersonate`, didn't use `gke-cluster`

> TODO

### D) Didn't use `gke-iam-impersonate`, didn't use `gke-cluster`

> TODO