package validation

import (
	"fmt"
	"golang.org/x/exp/maps"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var supportedEc2VolumeTypes = map[string]struct{}{
	"gp3": {},
	"io1": {},
	"io2": {},
}

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

func ValidEc2VolumeType() schema.SchemaValidateDiagFunc {
	return func(v interface{}, path cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if _, ok := supportedEc2VolumeTypes[value]; !ok {
			d := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("%q value is not supported. Should be one of: %v", value, maps.Keys(supportedEc2VolumeTypes)),
			}
			diags = append(diags, d)

		}
		return diags
	}
}
