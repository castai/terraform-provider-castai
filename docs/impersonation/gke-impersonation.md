# Impersonation for GKE clusters

Cast AI supports GKE service account impersonation as an alternative to using service account keys. This method enhances security by eliminating the need for key management while still allowing Cast AI to access your GCP resources.

For general information about this method, see https://docs.cast.ai/docs/sa-impersonation-gke.

This document describes how to configure GKE Impersonation using Terraform for new clusters and how to migrate existing clusters to Impersonation.

## History

Impersonation was originally supported via the `castai/iam-impersonate/gke` module and the `castai_gke_cluster_id` resource. This is now referred to as "Legacy Impersonation".

In November 2026, a new, simpler way to set up Impersonation was introduced. This new flow (referred to as "New Impersonation") came with the following changes:
- The `castai/iam-impersonate/gke` module was deprecated; it's no longer required to set up Impersonation.
- The `castai/gke-iam` module was enhanced to support Impersonation by specifying `authentication_method=impersonation`. It is now the preferred way to create the GCP resources needed for Impersonation.
- The `castai_gke_cluster_id` resource was deprecated; it's no longer required to set up Impersonation.
- The `castai_gke_cluster` resource was extended with the `target_service_account_email` and `castai_service_account_email` fields. These allow configuring Impersonation, taking the place of the `castai_gke_cluster_id` resource.

The New Impersonation relies on the **same cloud resources** (GCP Service Account and permissions to impersonate it) in your GCP Project, but passes the details of those to Cast AI using the `castai_gke_cluster` resource instead of `castai_gke_cluster_id`.

## What am I using now?

If you have a `google_service_account_key` resource in your Terraform state (called `castai_key` if you used the `castai/gke-iam` module), you're using Service Account Keys (static credentials).

If you have a `castai_gke_cluster_id` resource in your Terraform state (called `cluster_id` if you used the `castai/iam-impersonate/gke` module), you're using "Legacy Impersonation".

If none of the above apply, you're using "New Impersonation". You should see the `target_service_account_email` and `castai_service_account_email` fields set on your `castai_gke_cluster` resource.

## Connecting with Impersonation

If you want to connect a new cluster to Cast AI, the recommended way is "New Impersonation". Set `authentication_method = "impersonation"` on the `castai/gke-iam` module, then pass the `target_service_account_email` and `castai_service_account_email` values to the `castai/gke-cluster` module. You can check the [gke_impersonate provider example](https://github.com/castai/terraform-provider-castai/tree/master/examples/gke/gke_impersonate) for a complete example. Make sure you use the latest version of all modules.

## Migrating from Service Account Keys to Impersonation

If your cluster is connected using Service Account Keys (static credentials), you can migrate it to Impersonation. This involves the following three steps:
1. Set up GCP resources required for Impersonation to establish trust between your GCP Service Account and Cast's.
2. Switch your cluster to use the new authentication method.
3. Clean up GCP resources that are no longer required. Cast AI will no longer use your Service Account Keys, so they can be removed.

The following section describes some prerequisites, and the ones after that describe these steps in detail.

### Prerequisites

- Terraform **>= 1.9** if you use the `castai/gke-cluster` module (version `~> 11.1` requires it, and `terraform init` enforces it). If you manage the `castai_gke_cluster` resource directly, no particular Terraform version is required for this migration.
- Cast AI provider **>= 9.14.0**:
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
- If you're using the `castai/gke-cluster` module and have a lower version, update it to `~> 11.1`.

### 1. Setting up Impersonation GCP resources

If you **use** the `castai/gke-iam` module, update to version `0.6` and set the following arguments:

```terraform
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.6.0"

  # ...
  
  # Set these two arguments. Note, these are the defaults in 0.6.x. If you've
  # already updated to that version and haven't changed them, you're good. If
  # you've already set `authentication_method = "impersonation"`, you've
  # finished this migration. If you've set `authentication_method_migration =
  # "none"`, you've opted out of preparing impersonation; create the token
  # creator binding yourself, as shown below.
  authentication_method           = "service_account_key"
  authentication_method_migration = "prepare"
}
```

You should see the following changes:
- Adding `google_service_account_iam_member.castai_token_creator`. This allows the Cast AI Service Account to impersonate your Service Account.

If you **don't use** the `castai/gke-iam` module, you need to create a `google_service_account_iam_member` that allows the Cast AI Service Account to impersonate your Service Account:

```terraform
data "castai_impersonation_service_account" "gke" {
  provider_type = "gke"
}

resource "google_service_account_iam_member" "castai_token_creator" {
  # ID of the Service Account that you're currently creating a `google_service_account_key` for and passing that to Cast AI via the `castai_gke_cluster` resource:
  service_account_id = google_service_account.client_service_account.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${data.castai_impersonation_service_account.gke.email}"
}
```

### 2. Switch the authentication method

After verifying that the previous step was successfully applied and the `google_service_account_iam_member` resource was created, you can switch the authentication method from using the old Service Account Keys to Impersonation.

If you **use** the `castai/gke-cluster` module, you need to change its inputs like the following:

```terraform
module "castai_gke_cluster" {
  source  = "castai/gke-cluster/castai"
  version = "~> 11.1"

  # Remove gke_credentials:
  # gke_credentials = "{}"

  # Add the following if you use the `castai/gke-iam` module:
  target_service_account_email = module.castai_gke_iam.target_service_account_email
  castai_service_account_email = module.castai_gke_iam.castai_service_account_email
  
  # Add the following if you DON'T use the `castai/gke-iam` module:
  target_service_account_email = google_service_account.client_service_account.email
  castai_service_account_email = data.castai_impersonation_service_account.gke.email
}
```

If you **don't use** the `castai/gke-cluster` module, you need to change the arguments of the `castai_gke_cluster` resource like the following:

```terraform
resource "castai_gke_cluster" "castai_cluster" {
  # Remove credentials_json:
  # credentials_json = "{}"

  # Add the following if you use the `castai/gke-iam` module:
  target_service_account_email = module.castai_gke_iam.target_service_account_email
  castai_service_account_email = module.castai_gke_iam.castai_service_account_email

  # Add the following if you DON'T use the `castai/gke-iam` module:
  target_service_account_email = google_service_account.client_service_account.email
  castai_service_account_email = data.castai_impersonation_service_account.gke.email
}
```

### 3. Clean up Service Account Keys

At this point, your Service Account Keys are no longer in use by Cast AI, so you can remove them.

If you **use** the `castai/gke-iam` module, change the `authentication_method` argument and remove `authentication_method_migration`:

```terraform
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.6.0"

  # ...

  # Change this to `impersonation`:
  authentication_method = "impersonation"
  # Removing this is not strictly necessary, but the variable will be removed
  # in a future version of the module, so we recommend deleting it:
  # authentication_method_migration = "prepare"
}
```

If you **don't use** the `castai/gke-iam` module, remove the `google_service_account_key` resource:

```terraform
# Remove this resource:
# resource "google_service_account_key" "castai_key" {
#   service_account_id = google_service_account.client_service_account.name
#   # ...
# }
```

## Migrating from Legacy to New Impersonation

If your cluster is connected using Legacy Impersonation, you should migrate it to the New Impersonation. The essence of this is removing the `castai_gke_cluster_id` resource from Terraform state and passing the `target_service_account_email` and `castai_service_account_email` fields to the `castai_gke_cluster` resource instead. GCP resources should stay the same (as much as possible, see details below).

Depending on which Cast AI Terraform modules you relied on, the process is slightly different:

|                          | Used `iam-impersonate`                                                                                                       | Didn't use `iam-impersonate`                                                                                         |
|--------------------------|------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------|
| Used `gke-cluster`       | **A**. `iam-impersonate` module will be replaced with `gke-iam`.<br>The `gke-cluster` module receives different inputs.      | **B**. `castai_gke_cluster_id` resource will be removed.<br>The `gke-cluster` module receives different inputs.      |
| Didn't use `gke-cluster` | **C**. `iam-impersonate` module will be replaced with `gke-iam`.<br>`castai_gke_cluster` resource receives different inputs. | **D**. `castai_gke_cluster_id` resource will be removed.<br>`castai_gke_cluster` resource receives different inputs. |

The following section describes common prerequisites, and the ones after that show configuration examples for each of these cases.

### Prerequisites

- Terraform **>= 1.9** if you use the `castai/gke-cluster` module (version `~> 11.1` requires it, and `terraform init` enforces it). Otherwise **>= 1.7**: the examples below use `removed` blocks, which require it.
- Cast AI provider **>= 9.14.0**:
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
- If you're using the `castai/gke-cluster` module and have a lower version, update it to `~> 11.1`.
- If you're using the `castai/iam-impersonate/gke` module and have a lower version, update it to `1.1.0`.
- Keep the same `gke_cluster_name` through the migration: the service account name (`castai-gke-tf-<sha1 of gke_cluster_name>`) and the `castai_role` ID are derived from it. Changing it makes the `moved` blocks miss their targets.

### A) Used `iam-impersonate`, used `gke-cluster`

The following changes are required. Resources commented out can be deleted from the configuration, others need to be added or changed, see comments.

Changes need to be applied together in a single `terraform apply`.

```terraform
# 1. `castai/iam-impersonate/gke` module needs to be removed
#
# module "iam_impersonate" {
#   source  = "castai/iam-impersonate/gke"
#   version = "1.1.0"
#   # ...
# }

# It needs to be dropped from the state without deleting it
removed {
  from = module.iam_impersonate.castai_gke_cluster_id.cluster_id
  
  lifecycle {
    destroy = false
  }
}

# 2. `castai/gke-iam` needs to be added instead
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.6.0"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name

  authentication_method = "impersonation"
}

# Some resources can be moved to avoid recreating them
moved {
  from = module.iam_impersonate.google_service_account_iam_member.token_creator_binding
  to   = module.castai_gke_iam.google_service_account_iam_member.castai_token_creator[0]
}
moved {
  from = module.iam_impersonate.google_project_iam_member.service_account_user
  to   = module.castai_gke_iam.google_project_iam_member.project["roles/iam.serviceAccountUser"]
}
moved {
  from = module.iam_impersonate.google_project_iam_custom_role.castai_role
  to   = module.castai_gke_iam.google_project_iam_custom_role.castai_role
}
moved {
  from = module.iam_impersonate.google_service_account.client_service_account
  to   = module.castai_gke_iam.google_service_account.castai_service_account
}

# 3. `castai/gke-cluster` inputs need to be updated
module "castai_gke_cluster" {
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
- Destroying `module.iam_impersonate.time_sleep.wait_3_minutes`
- Destroying `module.iam_impersonate.google_service_account_iam_member.impersonation_user_binding` (this was never really necessary)
- Destroying `module.iam_impersonate.google_project_iam_binding.compute_manager_binding`
- Adding `module.castai_gke_iam.google_project_iam_member.project` (this takes the place of `module.iam_impersonate.google_project_iam_binding.compute_manager_binding`)
- Updating `module.castai_gke_cluster.castai_gke_cluster.castai_cluster` to remove `credentials_json` field
- Replacing (after moving) `module.castai_gke_iam.google_service_account_iam_member.castai_token_creator[0]` to remove an unnecessary `condition` (`title="AlwaysTrueCondition"`)
- Forgetting `module.iam_impersonate.castai_gke_cluster_id.cluster_id`
- Moving `module.iam_impersonate.google_project_iam_custom_role.castai_role`
- Moving `module.iam_impersonate.google_project_iam_member.service_account_user`
- Moving `module.iam_impersonate.google_service_account.client_service_account`

After applying these changes, you can increase the module versions at your discretion, but for the migration we suggest using these to make sure you see the same changes as above.

### B) Didn't use `iam-impersonate`, used `gke-cluster`

The following changes are required. Resources commented out can be deleted from the configuration, others need to be added or changed, see comments.

Changes need to be applied together in a single `terraform apply`.

```terraform
# 1. `castai_gke_cluster_id` resource needs to be removed
#
# resource "castai_gke_cluster_id" "cluster_id" {
#   client_service_account = google_service_account.client_service_account.email
#   # ...
# }

# It needs to be dropped from the state without deleting it
removed {
  from = castai_gke_cluster_id.cluster_id
  
  lifecycle {
    destroy = false
  }
}

# 2. `castai_impersonation_service_account` data source needs to be defined
data "castai_impersonation_service_account" "gke" {
  provider_type = "gke"
}

# 3. References to `castai_gke_cluster_id` need to be replaced with `castai_impersonation_service_account` in every `google_service_account_iam_member` where it was referenced:
resource "google_service_account_iam_member" "example_binding" {
  # Replace this:
  # member             = "serviceAccount:${castai_gke_cluster_id.cluster_id.cast_service_account}"
  # With this:
  # member             = "serviceAccount:${data.castai_impersonation_service_account.gke.email}"

  # ...
}

# 4. `castai/gke-cluster` inputs need to be updated
module "castai_gke_cluster" {
  source  = "castai/gke-cluster/castai"
  version = "~> 11.1"

  # Remove gke_credentials:
  # gke_credentials = "{}"

  # Add the following instead:
  target_service_account_email = google_service_account.client_service_account.email
  castai_service_account_email = data.castai_impersonation_service_account.gke.email
}
```

The expected changes in the resulting plan are:
- Updating `module.castai_gke_cluster.castai_gke_cluster.castai_cluster` to remove `credentials_json` field
- Forgetting `castai_gke_cluster_id.cluster_id`

After applying these changes, you can increase the module versions at your discretion, but for the migration we suggest using these to make sure you see the same changes as above.

### C) Used `iam-impersonate`, didn't use `gke-cluster`

The following changes are required. Resources commented out can be deleted from the configuration, others need to be added or changed, see comments.

Changes need to be applied together in a single `terraform apply`.

```terraform
# 1. `castai/iam-impersonate/gke` module needs to be removed
#
# module "iam_impersonate" {
#   source  = "castai/iam-impersonate/gke"
#   version = "1.1.0"
#   # ...
# }

# It needs to be dropped from the state without deleting it
removed {
  from = module.iam_impersonate.castai_gke_cluster_id.cluster_id
  
  lifecycle {
    destroy = false
  }
}

# 2. `castai/gke-iam` needs to be added instead
module "castai_gke_iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.6.0"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name

  authentication_method = "impersonation"
}

# Some resources can be moved to avoid recreating them
moved {
  from = module.iam_impersonate.google_service_account_iam_member.token_creator_binding
  to   = module.castai_gke_iam.google_service_account_iam_member.castai_token_creator[0]
}
moved {
  from = module.iam_impersonate.google_project_iam_member.service_account_user
  to   = module.castai_gke_iam.google_project_iam_member.project["roles/iam.serviceAccountUser"]
}
moved {
  from = module.iam_impersonate.google_project_iam_custom_role.castai_role
  to   = module.castai_gke_iam.google_project_iam_custom_role.castai_role
}
moved {
  from = module.iam_impersonate.google_service_account.client_service_account
  to   = module.castai_gke_iam.google_service_account.castai_service_account
}

# 3. `castai_gke_cluster` needs to be updated
resource "castai_gke_cluster" "castai_cluster" {
  # Remove credentials_json:
  # credentials_json = "{}"
  
  # Add the following instead:
  target_service_account_email = module.castai_gke_iam.target_service_account_email
  castai_service_account_email = module.castai_gke_iam.castai_service_account_email

  depends_on = [
    # This dependency (if it was present) can be removed; it's no longer necessary
    # Dependency on module.castai_gke_iam is implicit because of the new fields above:
    
    # module.iam_impersonate,
  ]
}
```

For the list of expected changes in the resulting plan, please see scenario **A)**. The only difference should be that `castai_gke_cluster` resource is not under the `castai_gke_cluster` module.

After applying these changes, you can increase the module versions at your discretion, but for the migration we suggest using these to make sure you see the same changes as above.

### D) Didn't use `iam-impersonate`, didn't use `gke-cluster`

The following changes are required. Resources commented out can be deleted from the configuration, others need to be added or changed, see comments.

Changes need to be applied together in a single `terraform apply`.

```terraform
# 1. `castai_gke_cluster_id` resource needs to be removed
#
# resource "castai_gke_cluster_id" "cluster_id" {
#   client_service_account = google_service_account.client_service_account.email
#   # ...
# }

# It needs to be dropped from the state without deleting it
removed {
  from = castai_gke_cluster_id.cluster_id
  
  lifecycle {
    destroy = false
  }
}

# 2. `castai_impersonation_service_account` data source needs to be defined
data "castai_impersonation_service_account" "gke" {
  provider_type = "gke"
}

# 3. References to `castai_gke_cluster_id` need to be replaced with `castai_impersonation_service_account` in every `google_service_account_iam_member` where it was referenced:
resource "google_service_account_iam_member" "example_binding" {
  # Replace this:
  # member             = "serviceAccount:${castai_gke_cluster_id.cluster_id.cast_service_account}"
  # With this:
  # member             = "serviceAccount:${data.castai_impersonation_service_account.gke.email}"

  # ...
}

# 4. `castai_gke_cluster` needs to be updated
resource "castai_gke_cluster" "castai_cluster" {
  # Remove credentials_json:
  # credentials_json = "{}"
  
  # Add the following instead:
  target_service_account_email = google_service_account.client_service_account.email
  castai_service_account_email = data.castai_impersonation_service_account.gke.email

  depends_on = [
    # This dependency (if it was present) can be removed; it's no longer necessary:
    
    # castai_gke_cluster_id.cluster_id,
  ]
}
```

The expected changes in the resulting plan are:
- Updating `castai_gke_cluster.castai_cluster` to remove `credentials_json` field
- Forgetting `castai_gke_cluster_id.cluster_id`
