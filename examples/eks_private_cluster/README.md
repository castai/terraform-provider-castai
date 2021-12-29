## EKS and CAST.AI setup example

Reference setup for EKS clusters without public API access, connected to CAST.AI

### Instruction for full setup (create vpc, eks and connect with cast.ai)

- `terraform apply -target module.vpc` (needs to be created before creating further iam and CAST.AI resources)
- `terraform apply` (applying the rest of resources)