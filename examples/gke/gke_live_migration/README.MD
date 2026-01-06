# Running LIVE on GKE

## Disclaimer

This is a temporary hack to enable early access to LIVE Migrations in Cast AI. We do not expect customers to run our installer in node configuration as we are doing in this example or having to make complicated setups.

This example contains a full creation of a GKE cluster and onboarding on Cast AI with the needed node init script to enable live migration in NEW created nodes.

## Network migration / TCP Migration

Currently not supported in GKE yet, be aware of the type of workloads you try to migrate for customers, it will fail eventually if they use persistent TCP connections.

# How to create your env
1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file with your project name, cluster name, cluster region and Cast AI API token.
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

# Notes

- Make sure to rebalance the cluster after it is fully onboarded, we replace the original image with a new image we know that contains a new version of containerd and is compatible with live.

# Closing

Enjoy, hack and provide feedback to live-team if you find any issue!
