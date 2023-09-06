## CAST AI example for managing organization members

Following example shows how to manage multiple organizations in CAST AI.

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file.
3. Run `terraform init`
4. Run `terraform apply apply -var-file=tf.vars`

## Importing already existing organization

This example can also be used to import an existing organization to Terraform.

1. `terraform import castai_organization_members.dev ORG_ID`
2. Observe state files and plan output.
3. Update the resource attributes to avoid the replacement.