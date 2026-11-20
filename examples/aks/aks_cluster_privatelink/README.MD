# Example of AKS cluster connected to CAST AI over Azure Private Link

Creates an AKS cluster with its resource group, VNet and subnets, connects the VNet to the CAST AI Private Link Service through a private endpoint, 
and onboards the cluster to CAST AI. From inside the cluster, every CAST AI call goes over the private endpoint - nothing else about the cluster is special. To make setup easier, the cluster itself is not fully private.

The private link bits are isolated in [privatelink.tf](privatelink.tf) - check that on the minimal sample setup that can be matched to existing cluster.

# Example setup

```
consumer subscription (this example)                 CAST AI producer subscription
┌─────────────────────────────────────┐              ┌───────────────────────────────┐
│ AKS (Azure CNI Overlay)             │              │ Private Link Service          │
│   10.20.0.0/20 nodes                │              │        │                      │
│        │                            │              │        ▼                      │
│        ▼                            │◀─ Private ──▶│ CAST AI services              │
│ Private Endpoint  10.20.16.0/24     │   Link       │                               │
│        │                            │              │                               │
│        ▼                            │              │                               │
│ Private DNS zone  cast.ai           │              │                               │
│   *.cast.ai → PE IP                 │              │                               │
└─────────────────────────────────────┘              └───────────────────────────────┘
```

Files, in the order to read them to fully understand the solution:

1. `vnet.tf` - resource group, VNet, node subnet and private endpoint subnet
2. `aks.tf` - AKS cluster (user-assigned identity, CNI Overlay, default egress)
3. `privatelink.tf` - **everything Private Link specific**: the private endpoint and the private DNS zone
4. `castai.tf` - CAST AI onboarding

## How the private path is chosen

No CAST AI component needs a private URL. The agents keep using the public
hostnames and the private DNS zone linked to the VNet is what makes them resolve
to the private endpoint IP:

| From | `api.cast.ai` resolves to | Path taken |
| --- | --- | --- |
| Pod in the cluster / any VM in the VNet | private endpoint IP in `10.20.16.0/24` | Private Link |
| Your machine running Terraform | public IP | Public API |

## Prerequisites

- `terraform`/`tofu` >= 1.3.2 and the Azure CLI logged in to the consumer subscription.
- A subscription on the CAST AI Private Link **auto-approval** list - otherwise the
  endpoint connection stays `Pending` and nothing resolves until someone approves
  it on the producer side.
- Rights to create a resource group, VNet, AKS cluster, private endpoint and
  private DNS zone, **plus** role assignments (`Owner` or `User Access Administrator`).
- A CAST AI API token for the environment the Private Link Service fronts.

## Use the example

```bash
cp terraform.tfvars.example terraform.tfvars
```

Fill in `subscription_id`, `cluster_name`, `cluster_region`, `castai_api_token`
and `private_link_service_alias`.

| Variable | Effect                                                                                   |
| --- |------------------------------------------------------------------------------------------|
| `private_link_service_alias` | Which producer deployment to connect to                                                  |
| `castai_api_url`, `castai_grpc_url`, `castai_api_grpc_url`, `castai_files_url`, `castai_kvisor_grpc_addr`, `castai_telemetry_url` | The CAST AI endpoints, exactly as in the docs per environment.                           |
| `castai_private_dns_zone` | Zone the wildcard record lives in. Keep it at `cast.ai` unless some DNS magic is needed. |

Then plan and deploy:

```bash
terraform init
terraform apply
```


Successful apply should have a cluster connected to CAST AI and display some data about the created resources.

## Verify

```bash
# 1. The private endpoint connection must be Approved
az network private-endpoint show --ids "$(terraform output -raw private_endpoint_id)" \
  --query "[privateLinkServiceConnections, manualPrivateLinkServiceConnections] | [] | [?privateLinkServiceConnectionState.status != null].privateLinkServiceConnectionState.status | [0]" \
  -o tsv
# Approved

# 2. Prove the private path from inside the cluster
az aks get-credentials --resource-group "$(terraform output -raw resource_group_name)" \
  --name "$(terraform output -raw aks_cluster_name)" --overwrite-existing
./scripts/verify.sh --context "$(terraform output -raw aks_cluster_name)"
```

The script starts a throwaway `ghcr.io/nicolaka/netshoot` pod in namespace
`castai-pls-verify` (deleted afterwards) and checks, from inside the cluster:

| # | Check | Expected |
| --- | --- | --- |
| 1 | `api.cast.ai`, `grpc.cast.ai` resolution | the private endpoint IP |
| 2 | `https://api.cast.ai/` | `remote_ip` = the private endpoint IP |
| 3 | TLS handshake to `api.cast.ai` | the real `api.cast.ai` certificate |
| 4 | Public DNS from the host (`dig`) | a different, public address |

It prints one `PASS`/`FAIL`/`SKIP` line per check and exits non-zero on any
`FAIL`, so it doubles as a smoke test. Knobs: `PE_IP=<ip>` states the expected
endpoint IP (otherwise read from `terraform output`), `KEEP_POD=1` leaves the pod
for poking, `--namespace` changes the namespace.

Sample output:
```
==> starting ghcr.io/nicolaka/netshoot:v0.16 pod in namespace castai-pls-verify (context test-context)
==> expecting private endpoint IP: 10.20.16.4
==> checking hostnames: api.eu.cast.ai grpc.eu.cast.ai api-grpc.eu.cast.ai files.eu.cast.ai kvisor.prod-eu.cast.ai telemetry.prod-eu.cast.ai 
PASS  api.eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  grpc.eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  api-grpc.eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  files.eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  kvisor.prod-eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  telemetry.prod-eu.cast.ai resolves to 10.20.16.4 (private endpoint)
PASS  https://api.eu.cast.ai/ used the private endpoint (remote_ip=10.20.16.4)
PASS  TLS through the endpoint serves the real api.eu.cast.ai certificate
PASS  public DNS still returns 141.101.90.96, so the private answer is an override

9 passed, 0 failed, 0 skipped
```

## Troubleshooting

| Symptom                                                  | Cause / fix                                                                                                                                                                              |
|----------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Connection state `Pending`                               | The subscription is not on the producer's auto-approval list; contact CAST AI                                                                                                            |
| `getent hosts <host>>` returns nothing in the cluster    | The zone is not linked to the VNet the cluster uses, or the hostname lives outside the zone - the wildcard only covers names under it, and unlisted names do not fall back to public DNS |
| `getent hosts` returns the public IP in the cluster      | The zone is not linked to the VNet the nodes use, or the zone name does not cover the hostname                                                                                           |
| TLS handshake resets / no certificate                    | The TLS handshake carries no server name (SNI). This is mandatory.                                                                                                                       |
| `azurerm_role_assignment` fails with `PrincipalNotFound` | Entra replication lag right after the identity is created; re-run `terraform apply`                                                                                                      |

## Cleanup

```bash
terraform destroy
```

Destroying the private endpoint removes the connection from the producer's
`privateEndpointConnections` list by itself; the producer's allow-list is untouched.