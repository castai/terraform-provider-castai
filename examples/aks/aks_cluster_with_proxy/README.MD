# AKS and CAST AI example where nodes require egress proxy configuration
Following example shows how to onboard AKS cluster to CAST AI, configure [Autoscaler policies](https://docs.cast.ai/reference/policiesapi_upsertclusterpolicies) and additional [Node Configurations](https://docs.cast.ai/docs/node-configuration/).

The AKS cluster has the following setup:
1. AKS cluster, a VNET and two firewalls are deployed in their own resource group.
2. AKS cluster is set up to use one of the firewalls for egress communication. 
3. Default communication for the cluster goes through a firewall that only allows traffic for AKS to work. No other egress traffic is allowed.
4. A second firewall is deployed in a peered VNET to the cluster's VNET. The firewall is in explicit proxy mode. This firewall allows egress traffic to CAST AI and Google storage (required by CAST).
5. The cluster is onboarded in CAST AI and configured so that the proxy firewall is used for egress communication. 

The goal of this example is to show how to configure CAST AI components so http proxy is enforced when communicating with the platform. 

Example configuration should be analysed in the following order:
1. Setup network topology - `network.tf`
2. Setup firewalls - `egress_firewall.tf` and `proxy_firewall.tf`
3. Create AKS cluster and configure it to use one firewall for egress - `aks.tf`
4. Create CAST AI related resources to connect AKS cluster to CAST AI and configure basic cluster components to use egress proxy - `castai.tf`.

# Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your cluster name, cluster region and CAST AI API token.
3. Initialize Terraform. Under example root folder run:
```
terraform init
```
4. Run Terraform apply:
```
terraform apply -var-file=tf.vars
```
5. To destroy resources created by this example:
```
terraform destroy -var-file=tf.vars
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
