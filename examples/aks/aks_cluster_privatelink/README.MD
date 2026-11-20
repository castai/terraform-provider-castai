# Example of AKS cluster connected to CAST AI over Azure Private Link

Creates an AKS cluster with its resource group, VNet and subnets, connects the
VNet to the CAST AI Private Link Service through a private endpoint, and onboards
the cluster to CAST AI. From inside the cluster, every CAST AI call goes over the
private endpoint - nothing else about the cluster is special.

```
consumer subscription (this example)                 CAST AI producer subscription
┌─────────────────────────────────────┐              ┌───────────────────────────────┐
│ AKS (Azure CNI Overlay)             │              │ Private Link Service          │
│   10.20.0.0/20 nodes                │              │   alias: <pls-alias>          │
│        │                            │              │        │                      │
│        ▼                            │◀─ Private ──▶│   NAT IPs                     │
│ Private Endpoint  10.20.16.0/24     │   Link       │        ▼                      │
│        │                            │              │ Internal LB :443 / :8080      │
│        ▼                            │              │        ▼                      │
│ Private DNS zone  cast.ai           │              │ nginx SNI router              │
│   api / api-grpc / grpc / files /   │              │   ├─▶ api.cast.ai             │
│   kvisor.prod-master → PE IP        │              │   └─▶ grpc.cast.ai …          │
└─────────────────────────────────────┘              └───────────────────────────────┘
```

Files, in the order to read them:

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

Both work at once - that is the point: Terraform onboards over the public API
while the cluster itself talks privately. TLS is passed through the producer's
SNI router untouched, so the certificates are the real CAST AI ones and nothing
needs to trust a proxy CA.

The example keeps the AKS default egress (`outbound_type = "loadBalancer"`) on
purpose: AKS needs internet access to bootstrap, and CAST AI images come from
`ghcr.io`, which the Private Link Service does not serve. A cluster without any
internet egress needs a different shape - private API server, `outbound_type =
"none"` and a private image registry. See `examples/aks/aks_cluster_with_proxy_privatelink`
for the variant that combines a proxy with Private Link.

## Prerequisites

- `terraform`/`tofu` >= 1.3.2 and the Azure CLI logged in to the consumer subscription.
- A subscription on the CAST AI Private Link **auto-approval** list - otherwise the
  endpoint connection stays `Pending` and nothing resolves until someone approves
  it on the producer side.
- Rights to create a resource group, VNet, AKS cluster, private endpoint and
  private DNS zone, **plus** role assignments (`Owner` or `User Access Administrator`).
- A CAST AI API token for the environment the Private Link Service fronts.

## Configure

```bash
cp terraform.tfvars.example terraform.tfvars
```

Fill in `subscription_id`, `cluster_name`, `cluster_region`, `castai_api_token`
and `private_link_service_alias` (read it in the producer workspace with
`tofu output -raw private_link_service_alias`).

The defaults target the US (`prod-master`) environment. The knobs that matter:

| Variable | Effect |
| --- | --- |
| `private_link_service_alias` | Which producer deployment to connect to |
| `castai_api_url`, `castai_grpc_url`, `castai_api_grpc_url`, `castai_files_url`, `castai_kvisor_grpc_addr`, `castai_telemetry_url` | The CAST AI endpoints, exactly as in the docs per environment. Defaults are prod-master (US); `terraform.tfvars.example` has ready-to-copy blocks for EU, India and dev-master |
| `castai_extra_dns_hostnames` | Extra FQDNs to point at the private endpoint, for a hostname with no URL variable |
| `castai_private_dns_zone` | Zone the records are created in. Keep it at `cast.ai` - record names are derived by stripping this suffix from the URL hostnames, and a precondition fails the apply if any hostname does not live under the zone |

The private DNS records are **derived from the URL variables**: the hostname is
parsed out of each URL and the zone suffix is stripped, so `api.eu.cast.ai`
becomes record `api.eu` and `kvisor.prod-eu.cast.ai` becomes `kvisor.prod-eu` -
no separate hostname list to keep in sync.

`terraform.tfvars.example` contains ready-to-copy URL blocks for the US, EU,
India and dev-master environments - switching is uncommenting one block.

## Deploy

```bash
terraform init
terraform apply
```

AKS dominates the apply time. Note `private_endpoint_ip` (in `10.20.16.0/24`) in:

```bash
terraform output
```

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
| 3 | TLS to the endpoint with SNI `api.cast.ai` | the real upstream `api.cast.ai` certificate |
| 4 | Public DNS from the host (`dig`) | a different, public address |

It prints one `PASS`/`FAIL`/`SKIP` line per check and exits non-zero on any
`FAIL`, so it doubles as a smoke test. Knobs: `PE_IP=<ip>` states the expected
endpoint IP (otherwise read from `terraform output`), `KEEP_POD=1` leaves the pod
for poking, `--namespace` changes the namespace.

The agents can also be checked directly:

```bash
kubectl -n castai-agent get pods
kubectl -n castai-agent exec deploy/castai-agent -- getent hosts api.cast.ai
# 10.20.16.x   api.cast.ai
```

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| Connection state `Pending` | The subscription is not on the producer's auto-approval list; nothing on the consumer side fixes it |
| `getent hosts api.cast.ai` returns nothing in the cluster | The hostname is not covered by the derived records - the zone is authoritative inside the VNet, unlisted names do not fall back to public DNS. Add it via `castai_dns_hostnames`. Azure Private DNS has no wildcard records |
| `getent hosts` returns the public IP in the cluster | The zone is not linked to the VNet the nodes use, or the zone name does not cover the hostname |
| TLS handshake resets / no certificate | The hostname is outside the producer's SNI allow-list, or the request has no SNI |
| `terraform apply` fails on subnet permissions | The caller lacks `Microsoft.Authorization/roleAssignments/write`; the cluster identity needs `Network Contributor` on the node subnet before AKS is created |
| `azurerm_role_assignment` fails with `PrincipalNotFound` | Entra replication lag right after the identity is created; re-run `terraform apply` |

## Cleanup

```bash
terraform destroy
```

Destroying the private endpoint removes the connection from the producer's
`privateEndpointConnections` list by itself; the producer's allow-list is untouched.

## Notes

- **Not for production:** Free AKS tier, an examples-only address plan, and no
  network policies.
- **Zone scope:** linking `cast.ai` makes the zone authoritative for every
  `*.cast.ai` name inside the VNet. Per-hostname zones (e.g. `api.cast.ai` with an
  apex A record) narrow that to one name each, at the cost of one zone link per
  hostname.
- The producer side (capacity limits, idle timers, allow-list management) is
  described in the CAST AI `cast-iac` repository, `privatelink/azure`.
