## EKS and CAST AI setup example

Reference setup for EKS clusters connected to CAST AI

### Instruction for full setup (create vpc, eks and connect with CAST AI)

- `terraform apply -target module.vpc` (needs to be created before creating further AWS IAM and CAST AI resources)
- `terraform apply` (applying the rest of resources)