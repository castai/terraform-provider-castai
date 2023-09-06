---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "castai_organization_members Resource - terraform-provider-castai"
subcategory: ""
description: |-
  CAST AI organization members resource to manage organization members
---

# castai_organization_members (Resource)

CAST AI organization members resource to manage organization members



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `organization_id` (String) CAST AI organization ID.

### Optional

- `members` (List of String) A list of email addresses corresponding to users who should be given member access to the organization.
- `owners` (List of String) A list of email addresses corresponding to users who should be given owner access to the organization.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `viewers` (List of String) A list of email addresses corresponding to users who should be given viewer access to the organization.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `update` (String)

