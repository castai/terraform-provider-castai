---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "castai_security_runtime_rule Resource - terraform-provider-castai"
subcategory: ""
description: |-
  Manages a CAST AI security runtime rule.
---

# castai_security_runtime_rule (Resource)

Manages a CAST AI security runtime rule.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Unique name of the runtime security rule. Name is used as resource identifier in Terraform.
- `rule_text` (String) CEL rule expression text.
- `severity` (String) Severity of the rule. One of SEVERITY_CRITICAL, SEVERITY_HIGH, SEVERITY_MEDIUM, SEVERITY_LOW, SEVERITY_NONE.

### Optional

- `category` (String) Category of the rule.
- `enabled` (Boolean) Whether the rule is enabled.
- `labels` (Map of String) Key-value labels attached to the rule.
- `resource_selector` (String) Optional CEL expression for resource selection.
- `rule_engine_type` (String) The engine type used to evaluate the rule. Only RULE_ENGINE_TYPE_CEL is currently supported.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `anomalies_count` (Number) Number of anomalies detected using this rule.
- `id` (String) The ID of this resource.
- `is_built_in` (Boolean) Indicates whether the rule is a built-in rule.
- `type` (String) Type of the rule (internal value).
- `used_custom_lists` (List of String) Custom lists used in this rule, if any.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `read` (String)


