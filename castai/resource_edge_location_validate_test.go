package castai

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// edgeLocationSchema returns the schema for the edge location resource.
func edgeLocationSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()
	ctx := context.Background()
	schemaResp := fwresource.SchemaResponse{}
	newEdgeLocationResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	return schemaResp
}

// edgeLocationSchemaObj returns the tftypes.Object representation of the edge
// location schema, used to build typed tftypes.Value instances in tests.
func edgeLocationSchemaObj(t *testing.T) tftypes.Object {
	t.Helper()
	ctx := context.Background()
	schemaResp := edgeLocationSchema(t)
	return schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
}

// makeEdgeLocationConfig builds a tfsdk.Config for the edge location resource.
// All fields not present in overrides are set to null.
func makeEdgeLocationConfig(t *testing.T, overrides map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	schemaResp := edgeLocationSchema(t)
	schemaObj := edgeLocationSchemaObj(t)

	fields := make(map[string]tftypes.Value, len(schemaObj.AttributeTypes))
	for name, typ := range schemaObj.AttributeTypes {
		fields[name] = tftypes.NewValue(typ, nil)
	}
	for name, val := range overrides {
		fields[name] = val
	}
	return tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: schemaObj.AttributeTypes}, fields),
		Schema: schemaResp.Schema,
	}
}

// allUnknownNestedAttr builds a tftypes.Value for a SingleNestedAttribute
// where all sub-fields are unknown, simulating values sourced from a resource
// that has not been created yet.
func allUnknownNestedAttr(t *testing.T, attrName string) tftypes.Value {
	t.Helper()
	schemaObj := edgeLocationSchemaObj(t)
	objType := schemaObj.AttributeTypes[attrName].(tftypes.Object)

	fields := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, typ := range objType.AttributeTypes {
		fields[k] = tftypes.NewValue(typ, tftypes.UnknownValue)
	}
	return tftypes.NewValue(objType, fields)
}

// invokeAttrObjectValidators calls all validator.Object validators declared on
// the named SingleNestedAttribute in the edge location schema. Attribute-level
// validators are NOT surfaced by resource.ConfigValidators(); they are invoked
// by the framework's ValidateResourceConfig RPC when iterating the schema.
func invokeAttrObjectValidators(
	t *testing.T,
	attrName string,
	cfg tfsdk.Config,
) validator.ObjectResponse {
	t.Helper()
	ctx := context.Background()
	schemaResp := edgeLocationSchema(t)

	attr, ok := schemaResp.Schema.Attributes[attrName]
	require.True(t, ok, "attribute %q not found in schema", attrName)

	sna, ok := attr.(rschema.SingleNestedAttribute)
	require.True(t, ok, "attribute %q is not a SingleNestedAttribute", attrName)

	// Extract the attribute's tftypes.Value from the raw config.
	rawFields := map[string]tftypes.Value{}
	require.NoError(t, cfg.Raw.As(&rawFields))
	attrTFVal := rawFields[attrName]

	// Convert to types.Object so it can be passed as ConfigValue.
	attrFrameworkVal, err := sna.GetType().ValueFromTerraform(ctx, attrTFVal)
	require.NoError(t, err, "failed to convert attribute value for %q", attrName)
	attrObj, ok := attrFrameworkVal.(types.Object)
	require.True(t, ok, "expected types.Object for attribute %q", attrName)

	resp := validator.ObjectResponse{}
	for _, v := range sna.ObjectValidators() {
		req := validator.ObjectRequest{
			Config:      cfg,
			ConfigValue: attrObj,
			Path:        path.Root(attrName),
		}
		v.ValidateObject(ctx, req, &resp)
	}
	return resp
}

// TestEdgeLocationResource_Validation_RegionRequired verifies that
// objectvalidator.AlsoRequires(region) fires when aws, gcp, or oci is set
// without a region, and does NOT fire when region is present or for custom.
func TestEdgeLocationResource_Validation_RegionRequired(t *testing.T) {
	t.Parallel()

	schemaObj := edgeLocationSchemaObj(t)

	// Build a minimal non-null custom object (the attribute has no sub-fields).
	customType := schemaObj.AttributeTypes["custom"].(tftypes.Object)
	customVal := tftypes.NewValue(customType, map[string]tftypes.Value{})

	tests := []struct {
		name        string
		attrName    string
		cfg         tfsdk.Config
		expectError bool
	}{
		{
			name:     "aws_without_region_errors",
			attrName: "aws",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"aws": allUnknownNestedAttr(t, "aws"),
				// "region" stays null
			}),
			expectError: true,
		},
		{
			name:     "gcp_without_region_errors",
			attrName: "gcp",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"gcp": allUnknownNestedAttr(t, "gcp"),
			}),
			expectError: true,
		},
		{
			name:     "oci_without_region_errors",
			attrName: "oci",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"oci": allUnknownNestedAttr(t, "oci"),
			}),
			expectError: true,
		},
		{
			name:     "custom_without_region_ok",
			attrName: "custom",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"custom": customVal,
				// "region" stays null — custom does not require it
			}),
			expectError: false,
		},
		{
			name:     "aws_with_region_ok",
			attrName: "aws",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"aws":    allUnknownNestedAttr(t, "aws"),
				"region": tftypes.NewValue(tftypes.String, "us-east-1"),
			}),
			expectError: false,
		},
		{
			name:     "gcp_with_region_ok",
			attrName: "gcp",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"gcp":    allUnknownNestedAttr(t, "gcp"),
				"region": tftypes.NewValue(tftypes.String, "us-central1"),
			}),
			expectError: false,
		},
		{
			name:     "oci_with_region_ok",
			attrName: "oci",
			cfg: makeEdgeLocationConfig(t, map[string]tftypes.Value{
				"oci":    allUnknownNestedAttr(t, "oci"),
				"region": tftypes.NewValue(tftypes.String, "us-phoenix-1"),
			}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := invokeAttrObjectValidators(t, tt.attrName, tt.cfg)
			if tt.expectError {
				assert.True(t, resp.Diagnostics.HasError(),
					"expected validation error for %q but got none", tt.attrName)
			} else {
				assert.False(t, resp.Diagnostics.HasError(),
					"unexpected validation error for %q: %v", tt.attrName, resp.Diagnostics)
			}
		})
	}
}

// TestEdgeLocationResource_Validation_UnknownFieldsDoNotCauseSpuriousErrors is
// a regression test for the bug introduced in commit f5018c0 where
// ValidateResource called req.Config.Get(ctx, &edgeLocationModel{}) to read
// the whole config. That fails at plan time when any attribute is unknown
// (e.g., zones or organization_id referencing another resource's output),
// because Go slice/struct types cannot represent Terraform's "unknown" state.
//
// The fix replaced the whole-model decode with objectvalidator.AlsoRequires on
// the aws, gcp and oci schema attributes. That validator reads only the
// specific attributes it needs and handles unknowns natively.
func TestEdgeLocationResource_Validation_UnknownFieldsDoNotCauseSpuriousErrors(t *testing.T) {
	t.Parallel()

	schemaObj := edgeLocationSchemaObj(t)

	// zones is a ListNestedAttribute. Mark the entire list as unknown, which
	// simulates a user passing `zones = some_resource.zones` where the source
	// resource has not been applied yet.
	// A []zoneModel decode (the old approach) would fail here.
	zonesUnknown := tftypes.NewValue(schemaObj.AttributeTypes["zones"], tftypes.UnknownValue)

	cfg := makeEdgeLocationConfig(t, map[string]tftypes.Value{
		// aws is set but all its subfields are unknown.
		"aws": allUnknownNestedAttr(t, "aws"),
		// region is a known value — the AlsoRequires validator should pass.
		"region": tftypes.NewValue(tftypes.String, "us-east-1"),
		// zones is entirely unknown.
		"zones": zonesUnknown,
		// organization_id / cluster_id are unknown (from another resource).
		"organization_id": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"cluster_id":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	resp := invokeAttrObjectValidators(t, "aws", cfg)
	assert.False(t, resp.Diagnostics.HasError(),
		"validators must not error when config contains unknown values: %v", resp.Diagnostics)
}
