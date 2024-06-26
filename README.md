<a href="https://cast.ai">
    <img src="https://cast.ai/wp-content/themes/cast/img/cast-logo-dark-blue.svg" align="right" height="100" />
</a>

Terraform Provider for CAST AI
==================


Website: https://www.cast.ai

[![Build Status](https://github.com/castai/terraform-provider-castai/workflows/Build/badge.svg)](https://github.com/castai/terraform-provider-castai/actions)



Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.13+
- [Go](https://golang.org/doc/install) 1.19 (to build the provider plugin)

Using the provider
----------------------

To install this provider, put the following code into your Terraform configuration. Then, run `terraform init`.

```terraform
terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = "2.0.0" # can be omitted for the latest version
    }
  }
  required_version = ">= 0.13"
}

provider "castai" {
  api_token = "<<your-castai-api-key>>"
}
```

Alternatively, you can pass api key via environment variable:

```sh
$ CASTAI_API_TOKEN=<<your-castai-api-key>> terraform plan
```

For more logs use the log level flag:

```sh
$ TF_LOG=DEBUG terraform plan
```

More examples can be found [here](examples/).

_Learn why `required_providers` block is required
in [terraform 0.13 upgrade guide](https://www.terraform.io/upgrade-guides/0-13.html#explicit-provider-source-locations)
._

Migrating to 1.x.x
------------
Version 1.x.x no longer supports setting cluster configuration directly and `castai_node_configuration` resource should
be used. This applies to all `castai_*_cluster` resources.

Additionally, in case of `castai_eks_cluster` `access_key_id` and `secret_access_key` was removed in favor of `assume_role_arn`.

Having old configuration:

```terraform
resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name

  access_key_id     = var.aws_access_key_id
  secret_access_key = var.aws_secret_access_key

  subnets              = module.vpc.private_subnets
  dns_cluster_ip       = "10.100.0.10"
  instance_profile_arn = var.instance_profile_arn
  security_groups      = [aws_security_group.test.id]
}
```

New configuration will look like:

```terraform
resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name

  assume_role_arn = var.assume_role_arn
}

resource "castai_node_configuration" "test" {
  name       = "default"
  cluster_id = castai_eks_cluster.this.id
  subnets    = module.vpc.private_subnets
  eks {
    instance_profile_arn = var.instance_profile_arn
    dns_cluster_ip       = "10.100.0.10"
    security_groups      = [aws_security_group.test.id]
  }
}

resource "castai_node_configuration_default" "test" {
  cluster_id       = castai_eks_cluster.test.id
  configuration_id = castai_node_configuration.test.id
}
```

If you have used `castai-eks-cluster` module follow:
https://github.com/castai/terraform-castai-eks-cluster/blob/main/README.md#migrating-from-2xx-to-3xx


Migrating from 3.x.x to 4.x.x
---------------------------

Version 4.x.x changed:
* `castai_eks_clusterid` type from data source to resource

Having old configuration: 

```terraform
data "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
} 
```
and usage `data.castai_eks_clusterid.cluster_id.id`

New configuration will look like:

```terrafrom
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}
```
and usage `castai_eks_clusterid.cluster_id.id`

* removal of `castai_cluster_token` resource in favour of `cluster_token` in `castai_eks_cluster`

Having old configuration: 
```terraform
resource "castai_cluster_token" "this" {
  cluster_id = castai_eks_cluster.this.id
}
resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name
}
    
```
and usage `castai_cluster_token.this.cluster_token`

New configuration will look like:
```terraform
resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name
}
```
and usage `castai_eks_cluster.this.cluster_token`

* default value for `imds_v1` was change to `true`, in case that your configuration didn't had this specified
please explicitly set this value to `false`

Migrating from 4.x.x to 5.x.x
---------------------------

Version 5.x.x changed:
* Terraform provider adopts [default node template concept](https://docs.cast.ai/docs/default-node-template)
* Removed `spotInstances` field from `autoscaler_policies_json` attribute in `castai_autoscaler_policies` resource
* Removed `customInstancesEnabled` field from `autoscaler_policies_json` attribute in `castai_autoscaler_policies` resource
* Removed `nodeConstraints` field from `autoscaler_policies_json` attribute in `castai_autoscaler_policies` resource
* All valid fields which were removed from `autoscaler_policies_json` have mapping in `castai_node_template` [resource](https://registry.terraform.io/providers/CastAI/castai/latest/docs/resources/node_template)

Old configuration:
```terraform
resource "castai_autoscaler" "castai_autoscaler_policies" {
  cluster_id               = data.castai_eks_clusterid.cluster_id.id // or other reference

  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true,
            "customInstancesEnabled": true,
            "nodeConstraints": {
                "enabled": true,
                "minCpuCores": 2,
                "maxCpuCores": 4,
                "minRamMib": 3814,
                "maxRamMib": 16384
            }
        },
        "spotInstances": {
            "enabled": true,
            "clouds": ["gcp"],
            "spotBackups": {
                "enabled": true
            }
        },
        "nodeDownscaler": {
            "enabled": true,
            "emptyNodes": {
                "enabled": true
            },
            "evictor": {
                "aggressiveMode": true,
                "cycleInterval": "5m10s",
                "dryRun": false,
                "enabled": true,
                "nodeGracePeriodMinutes": 10,
                "scopedMode": false
            }
        }
    }
  EOT
}
```

New configuration:
```terraform
resource "castai_autoscaler" "castai_autoscaler_policies" {
  cluster_id               = data.castai_eks_clusterid.cluster_id.id // or other reference

  autoscaler_policies_json = <<-EOT
    {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true
        },
        "nodeDownscaler": {
            "enabled": true,
            "emptyNodes": {
                "enabled": true
            },
            "evictor": {
                "aggressiveMode": true,
                "cycleInterval": "5m10s",
                "dryRun": false,
                "enabled": true,
                "nodeGracePeriodMinutes": 10,
                "scopedMode": false
            }
        }
    }
  EOT
}

resource "castai_node_template" "default_by_castai" {
  cluster_id = data.castai_eks_clusterid.cluster_id.id // or other reference

  name                     = "default-by-castai"
  configuration_id         = castai_node_configuration.default.id // or other reference
  is_default               = true
  should_taint             = false
  custom_instances_enabled = true

  constraints {
    architectures = [
      "amd64",
      "arm64",
    ]
    on_demand          = true
    spot               = true
    use_spot_fallbacks = true
    min_cpu            = 2
    max_cpu            = 4
    min_memory         = 3814
    max_memory         = 16384
  }

  depends_on = [ castai_autoscaler.castai_autoscaler_policies ]
}
```

If you have used `castai-eks-cluster` or other modules follow:
https://github.com/castai/terraform-castai-eks-cluster/blob/main/README.md#migrating-from-5xx-to-6xx

Note: `default-by-castai` default node template is created in background by CAST.ai, when creating managed resource
in Terraform the provider will handle create as update. **Importing `default-by-castai` default node template into Terraform
state is not needed if you follow the migration guide**. Despite not being needed it can be performed and everything
will work correctly.

Example of node template import:
```sh
terraform import castai_node_template.default_by_castai 105e6fa3-20b1-424e-v589-9a64d1eeabea/default-by-castai
```

Migrating from 5.x.x to 6.x.x
---------------------------

Version 6.x.x changed:
* Removed `custom_label` attribute in `castai_node_template` resource. Use `custom_labels` instead.

Old configuration:
```terraform
module "castai-aks-cluster" {
  node_templates = {
    spot_tmpl = {
      custom_label = {
        key = "custom-label-key-1"
        value = "custom-label-value-1"
      }
    }
  }
}
```

New configuration:
```terraform
module "castai-aks-cluster" {
  node_templates = {
    spot_tmpl = {
      custom_labels = {
        custom-label-key-1 = "custom-label-value-1"
      }
    }
  }
}
```
For more information for `castai-aks-cluster` module follow:
https://github.com/castai/terraform-castai-aks/blob/main/README.md#migrating-from-2xx-to-3xx
If you have used `castai-eks-cluster` or other modules follow:
https://github.com/castai/terraform-castai-eks-cluster/blob/main/README.md#migrating-from-6xx-to-7xx
If you have used `castai-gke-cluster` or other modules follow:
https://github.com/castai/terraform-castai-gke-cluster/blob/main/README.md#migrating-from-3xx-to-4xx


Migrating from 6.x.x to 7.x.x
---------------------------

Version 7.x.x changed:
* Removed `compute_optimized` and `storage_optimized` attributes in `castai_node_template` resource, `constraints` object. Use `compute_optimized_state` and `storage_optimized_state` instead.

Old configuration:
```terraform
module "castai-aks-cluster" {
  node_templates = {
    spot_tmpl = {
      constraints = {
        compute_optimized = false
        storage_optimized = true
      }
    }
  }
}
```

New configuration:
```terraform
module "castai-aks-cluster" {
  node_templates = {
    spot_tmpl = {
      constraints = {
        compute_optimized_state = "disabled"
        storage_optimized_state = "enabled"
      }
    }
  }
}
```

* [v7.4.X] Deprecated `autoscaler_policies_json` attribute in `castai_autoscaler` resource. Use `autoscaler_settings` instead.

Old configuration:
```hcl
resource "castai_autoscaler" "castai_autoscaler_policies" {
  cluster_id               = data.castai_eks_clusterid.cluster_id.id // or other reference
  
  autoscaler_policies_json = <<-EOT
     {
        "enabled": true,
        "unschedulablePods": {
            "enabled": true
        },
        "nodeDownscaler": {
            "enabled": true,
            "emptyNodes": {
                "enabled": true
            },
            "evictor": {
                "aggressiveMode": false,
                "cycleInterval": "5m10s",
                "dryRun": false,
                "enabled": true,
                "nodeGracePeriodMinutes": 10,
                "scopedMode": false
            }
        },
        "nodeTemplatesPartialMatchingEnabled": false,
        "clusterLimits": {
            "cpu": {
                "maxCores": 20,
                "minCores": 1
            },
            "enabled": true
        }
    }
  EOT
}
```

New configuration:
```hcl
resource "castai_autoscaler" "castai_autoscaler_policies" {
  cluster_id               = data.castai_eks_clusterid.cluster_id.id // or other reference

  autoscaler_settings {
    enabled = true
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled = false
      }

      evictor {
        aggressive_mode           = false
        cycle_interval            = "5m10s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits {
      enabled = true
      
      cpu {
        max_cores = 20
        min_cores = 1
      }
    }
  }
}
```

For more information for `castai-aks-cluster` module follow:
https://github.com/castai/terraform-castai-aks/blob/main/README.md#migrating-from-3xx-to-4xx
If you have used `castai-eks-cluster` or other modules follow:
https://github.com/castai/terraform-castai-eks-cluster/blob/main/README.md#migrating-from-7xx-to-8xx
If you have used `castai-gke-cluster` or other modules follow:
https://github.com/castai/terraform-castai-gke-cluster/blob/main/README.md#migrating-from-4xx-to-5xx


Developing the provider
---------------------------

Make sure you have [Go](http://www.golang.org) installed on your machine (please check
the [requirements](#requirements)).

To build the provider locally:

```sh
$ git clone https://github.com/CastAI/terraform-provider-castai.git
$ cd terraform-provider-castai
$ make build
```

After you build the provider, you have to set the `~/.terraformrc` configuration to let terraform know you want to use local provider:
```terraform
provider_installation {
  dev_overrides {
    "castai/castai" = "<path-to-terraform-provider-castai-repository>"
  }
  direct {}
}
```

_`make build` builds the provider and install symlinks to that build for all terraform projects in `examples/*` dir.
Now you can work on `examples/localdev`._

Whenever you make changes to the provider re-run `make build`.

You'll need to run `terraform init` in your terraform project again since the binary has changed.

To run unit tests:

```sh
$ make test
```

Releasing the provider
----------------------

This repository contains a github action to automatically build and publish assets for release when
tag is pushed with pattern `v*` (ie. `v0.1.0`).

[Gorelaser](https://goreleaser.com/) is used to produce build artifacts matching
the [layout required](https://www.terraform.io/docs/registry/providers/publishing.html#manually-preparing-a-release)
to publish the provider in the Terraform Registry.

Releases will appear as **drafts**. Once marked as published on the GitHub Releases page, they will become available via
the Terraform Registry.
