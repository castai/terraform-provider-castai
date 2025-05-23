---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "castai_service_account Resource - terraform-provider-castai"
subcategory: ""
description: |-
  Service account resource allows managing CAST AI service accounts.
---

# castai_service_account (Resource)

Service account resource allows managing CAST AI service accounts.

## Example Usage

```terraform
resource "castai_service_account" "service_account" {
  organization_id = organization.id
  name            = "example-service-account"
  description     = "service account description"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the service account.
- `organization_id` (String) ID of the organization.

### Optional

- `description` (String) Description of the service account.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `author` (List of Object) Author of the service account. (see [below for nested schema](#nestedatt--author))
- `email` (String) Email of the service account.
- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `read` (String)
- `update` (String)


<a id="nestedatt--author"></a>
### Nested Schema for `author`

Read-Only:

- `email` (String)
- `id` (String)
- `kind` (String)


