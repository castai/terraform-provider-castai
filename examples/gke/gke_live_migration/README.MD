# Running LIVE on GKE

This example creates a GKE cluster, onboards it to Cast AI, installs the `castai-live` Helm chart and creates a CLM-enabled node template.

Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create GKE cluster - `gke.tf`
3. Connect the cluster to Cast AI, configure Autoscaler, Node Configurations and Node Templates - `castai.tf`

# How to create your env
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your project name, cluster name, cluster region and Cast AI API token.

| Variable | Description |
| --- | --- |
| `live_helm_version` | Optional. Version of the `castai-live` Helm chart. Latest when not set. |
| `install_helm_live` | Optional. Set to `false` to skip the `castai-live` Helm chart (developers only). Default `true`. |

3. Initialize tofu. Under example root folder run:
```bash
tofu init
```
4. Run tofu apply:
```
tofu apply
```
5. To destroy resources created by this example:
```
tofu destroy
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting

# CLM Configuration

CLM is enabled by having:
1. `install_live = true` (`install_live = var.install_helm_live`, default value is `true`) in the CAST AI module configuration (`module "castai-gke-cluster"`). It installs the `castai-live` Helm chart.
2. A dedicated `live` node configuration in the CAST AI module configuration.
3. `clm_enabled = true` in the `live_tmpl` node template (`node_templates` in the CAST AI module configuration) that uses the `live` node configuration.
4. `clm_networking_mode = "tc"` in the `live_tmpl` node template. Refer to [Traffic Control (TC)](https://docs.cast.ai/docs/clm-reference-traffic-control).
All are in the `castai.tf` file.

Notes:
- The instance families of the `live_tmpl` node template are limited to `c2d`. Refer to [CLM requirements and limitations](https://docs.cast.ai/docs/clm-requirements-and-limitations).
- The `live_tmpl` node template has `should_taint = true`. Only pods that tolerate the `scheduling.cast.ai/node-template=live-tmpl:NoSchedule` taint are scheduled on its nodes.

All other GKE/CAST AI resources mostly manage the GKE cluster and its connection to CAST AI, not specifically the CLM feature.

# Closing

Enjoy, hack and provide feedback to live-team if you find any issue!
