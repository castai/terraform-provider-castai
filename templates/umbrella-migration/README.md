# Migrating to Cast AI umbrella Helm chart

This document guides you in migrating from standalone Cast AI Helm charts to the
[Cast AI umbrella Helm chart](https://docs.cast.ai/docs/helm-charts#umbrella-chart). It applies to you if you are using
the Cast AI cluster Terraform modules to install Cast AI Helm charts:

- [castai-eks-cluster](https://github.com/castai/terraform-castai-eks-cluster)
- [castai-gke-cluster](https://github.com/castai/terraform-castai-gke-cluster)
- [castai-aks](https://github.com/castai/terraform-castai-aks)

**If you're not using these modules** for installing Cast AI Helm charts, follow this guide instead:
https://docs.cast.ai/docs/castctl-migrate-gitops

## Overview

The migration to the umbrella chart involves removing the standalone Cast AI Helm charts from your cluster and
replacing them with the umbrella chart. There are two ways of achieving this:
- **Option A: Terraform only**. In this case, you'll use Terraform to uninstall the Helm releases and install the umbrella
  chart. Everything is handled by Terraform, there are no manual steps required, but there's a brief downtime (3-5
  minutes) in all Cast AI services between uninstalling and installing the Helm releases.
- **Option B: With manual access**. In this case, you'll use [castctl](https://docs.cast.ai/docs/connect-with-castctl) to
  clean up the state of standalone Helm releases from your cluster, while leaving the Cast AI components running. Then
  you install the umbrella chart using Terraform. This option relies on executing `castctl` commands with access to the
  Kubernetes cluster.

If you can plan for the brief outage in Cast AI services, the **recommended approach is Option A**. If you can access
your cluster with `castctl`, you can choose Option B with no downtime instead.

## Prerequisites

1. Make sure you have no [scheduled rebalancing](https://docs.cast.ai/docs/scheduled-rebalancing) planned during the 
   migration.
2. If you're using [Container Live Migration](https://docs.cast.ai/docs/clm-overview), you need to disable Evictor, by
   setting `autoscaler_settings.node_downscaler.evictor=false` in the cluster Terraform module's variables.

These changes **need to be applied before proceeding** with the migration.

## Migration

### Option A: Terraform only

The migration happens in two phases as two separate Terraform runs.

#### Phase 1: Upgrading the cluster module and keeping CRDs

Implement the following changes in your Terraform configuration:

1. Update the version of the cluster module according to [Appendix: Module versions](#appendix-module-versions)
2. Set `workload_autoscaler_keep_crds=true` in the module's inputs

Example configuration:

```terraform
module "castai_eks_cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 15.0" # <- Look up version below in "Appendix: Module versions"
  # ...
  workload_autoscaler_keep_crds = true # <- Add this line
}
```

When applying, the following changes are expected in the Terraform plan:
- `helm_release.castai_workload_autoscaler` will include `preDeleteHook.enabled=false` and `crds.keep=true` settings.
- `helm_release`s might be updated to newer versions.
- Some `helm_release`s might show up as being `moved`.

**Apply these changes** before continuing with the next phase.

#### Phase 2: Replacing Helm releases with umbrella chart

Implement the following change in your Terraform configuration:

1. Set `umbrella_enabled=true` in the module's inputs

Example configuration:

```terraform
module "castai_eks_cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 15.0"
  # ...
  workload_autoscaler_keep_crds = true
  umbrella_enabled              = true # <- Add this line
}
```

When applying, the following changes are expected in the Terraform plan:
- Some existing `helm_release`s being deleted. Depending on your settings, the following `helm_release`
  resources _may_ be deleted:
  - `castai_agent`
  - `castai_cluster_controller`
  - `castai_cluster_controller_self_managed`
  - `castai_evictor`
  - `castai_evictor_self_managed`
  - `castai_evictor_ext`
  - `castai_kvisor`
  - `castai_kvisor_self_managed`
  - `castai_live`
  - `castai_live_self_managed`
  - `castai_pod_mutator`
  - `castai_pod_mutator_self_managed`
  - `castai_pod_pinner`
  - `castai_pod_pinner_self_managed`
  - `castai_spot_handler`
  - `castai_spot_handler_self_managed`
  - `castai_workload_autoscaler`
  - `castai_workload_autoscaler_self_managed`
  - `castai_workload_autoscaler_exporter`
  - `castai_workload_autoscaler_exporter_self_managed`
- The `castai_umbrella_cast_managed` or `castai_umbrella_self_managed` `helm_release` being created.

Due to how `depends_on` relations are set up inside the modules, the deletions will all complete before the
installation of the umbrella chart begins.

**Apply these changes** to conclude the migration. Keep in mind that during this process, Cast AI services will briefly
be unavailable while components are uninstalled and installed again.

### Option B: With manual access

The migration happens in three phases:
1. A `castctl` command will remove Helm release state from your cluster. 
2. A Terraform run installs the umbrella chart adopting the existing resources.
3. Another `castctl` command cleans up orphaned resources.

Between phase 2 and 3, some components will run duplicated in the cluster. This is expected, but might require
additional cluster resources (CPU and memory). If you're using Cast AI Node Autoscaling, you should be covered, but
otherwise you might have to accommodate for the additional components.

Before starting the migration, confirm that you have [castctl](https://docs.cast.ai/docs/connect-with-castctl) installed
with version `0.15.0` or newer:

```shell
# This should print "0.15.0" or newer:
castctl version
```

#### Phase 1: Remove Helm release states

Run the following command to remove Cast AI Helm release `secrets` from the cluster. It will ask for confirmation. Make
sure you **verify the cluster name** you're executing against:

```shell
castctl cluster migrate pre-umbrella-install
```

Confirm the command and wait for it to execute before continuing with the next phase.

#### Phase 2: Install the umbrella chart with Terraform

Implement the following changes in your Terraform configuration:

1. Update the version of the cluster module according to [Appendix: Module versions](#appendix-module-versions)
2. Set `workload_autoscaler_keep_crds=true` in the module's inputs
3. Set `umbrella_enabled=true` in the module's inputs

Example configuration:

```terraform
module "castai_eks_cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 15.0" # <- Look up version below in "Appendix: Module versions"
  # ...
  workload_autoscaler_keep_crds = true # <- Add this line
  umbrella_enabled              = true # <- Add this line
}
```

When applying, the following changes are expected in the Terraform plan:
- The `castai_umbrella_cast_managed` or `castai_umbrella_self_managed` `helm_release` being created.

**Apply these changes**.

#### Phase 3: Clean up orphaned resources

In the first phase, some Pods and ReplicaSets became orphaned. The following `castctl` command will remove these. It
will ask for confirmation. Make sure you **verify the cluster name** you're executing against:

```shell
castctl cluster migrate post-umbrella-install
```

This command concludes the migration.

## Post-migration

If you've changed rebalancing schedules or Evictor settings during the prerequisites, you can now revert those changes.

## Appendix: Module versions

The following major version of the cluster modules needs to be used during the migration. **Do NOT use a newer major
version!**

| Module               | Major version | Recommended version constraint |
|----------------------|---------------|--------------------------------|
| `castai-eks-cluster` | 15            | `~> 15.0`                      |
| `castai-gke-cluster` | 11            | `~> 11.0`                      |
| `castai-aks`         | 12            | `~> 12.0`                      |