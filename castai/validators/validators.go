package validators

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = Base64Validator{}

// Base64Validator validates that a string is valid base64.
type Base64Validator struct{}

func ValidBase64() Base64Validator {
	return Base64Validator{}
}

func (v Base64Validator) Description(ctx context.Context) string {
	return "value must be valid base64 encoded string"
}

func (v Base64Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v Base64Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Base64 Value",
			fmt.Sprintf("Value must be valid base64 encoded string: %s", err.Error()),
		)
	}
}
