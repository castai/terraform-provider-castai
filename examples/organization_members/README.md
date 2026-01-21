## CAST AI example for managing organization members

Following example shows how to manage multiple organizations in CAST AI.

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update `terraform.tfvars` file.
3. Run `terraform init`
4. Run `terraform apply`

## Importing already existing organization

This example can also be used to import an existing organization to Terraform.

1. `terraform import castai_organization_members.dev ORG_ID`
2. Observe state files and plan output.
3. Update the resource attributes to avoid the replacement.