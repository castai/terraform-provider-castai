package validation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ValidKeyPairFormat() schema.SchemaValidateDiagFunc {
	return func(v interface{}, path cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if !strings.HasPrefix(value, "key-") {
			d := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("%q should start with 'key-'", value),
			}
			diags = append(diags, d)
		}
		return diags
	}
}
