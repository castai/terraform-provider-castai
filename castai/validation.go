package castai

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// terraform plugin sdk v2 deprecates ValidateFunc (in favor of ValidateDiagFunc) but only has validation helpers for ValidateFunc
// temporary wrapper taken from the PR https://github.com/hashicorp/terraform-plugin-sdk/pull/611
func toDiagFunc(validator schema.SchemaValidateFunc) schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		attr := p[len(p)-1].(cty.GetAttrStep)
		ws, es := validator(i, attr.Name)

		for _, w := range ws {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       w,
				AttributePath: p,
			})
		}
		for _, e := range es {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       e.Error(),
				AttributePath: p,
			})
		}
		return diags
	}
}
