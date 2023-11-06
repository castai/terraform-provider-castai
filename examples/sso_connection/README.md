## CAST AI example for creating SSO connection

Following example shows how setup Azure AD to create SSO trust relationship with CAST AI.

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file.
3. Run `terraform init`
4. Run `terraform apply apply -var-file=tf.vars`