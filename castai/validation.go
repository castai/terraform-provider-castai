package castai

import (
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func validateRFC3339TimeOrEmpty(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok || v == "" {
		return nil // Allow empty strings without error
	}

	_, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid expires_at format",
				Detail:   "The expires_at field must be in RFC3339 format or an empty string.",
			},
		}
	}

	return nil
}
