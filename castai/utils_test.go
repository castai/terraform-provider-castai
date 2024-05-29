package castai

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func Test_normalizeJSON(t *testing.T) {
	tests := map[string]struct {
		jsonOne string
		jsonTwo string
	}{
		"should produce the same json when key order is different in inputs": {
			jsonOne: `{
				"second": "value",
				"first": "value"
			}`,
			jsonTwo: `{
				"first": "value",
				"second": "value"
			}`,
		},
		"should produce the same json when key order is different in nested inputs": {
			jsonOne: `{
				"second": "value",
				"first": "value",
				"nested": {
					"second": "value",
					"first": "value"
				}
			}`,
			jsonTwo: `{
				"first": "value",
				"nested": {
					"first": "value",
					"second": "value"
				},
				"second": "value"
			}`,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			r := require.New(t)

			jsonOneBytes := []byte(tt.jsonOne)
			jsonTwoBytes := []byte(tt.jsonTwo)

			resultOne, err := normalizeJSON(jsonOneBytes)
			r.NoError(err)

			resultTwo, err := normalizeJSON(jsonTwoBytes)
			r.NoError(err)

			r.Equal(string(resultOne), string(resultTwo))
		})
	}
}

func Test_ExtractNestedValues(t *testing.T) {
	tt := []struct {
		name                 string    // the name of the test
		key                  string    // the key to extract from the data
		data                 cty.Value // the data for testResource() resource state
		unwrapSingleItemList bool      // if true, single item lists will be unwrapped
		includeZeroValues    bool      // if true, zero values will be included in the result
		shouldExist          bool      // if true, the key should exist in the data and we should get a result
		expectedData         any       // the expected result
	}{
		{
			name:              "when data is empty and includeZeroValues true, than should return exist flag false",
			data:              cty.NilVal,
			includeZeroValues: true,
			shouldExist:       false,
		},
		{
			name:              "when data is empty and includeZeroValues false, than should return exist flag false",
			data:              cty.NilVal,
			includeZeroValues: false,
			shouldExist:       false,
		},
		{
			name:        "when key is not present in data, than should return exist flag false",
			key:         "foo",
			data:        cty.ObjectVal(map[string]cty.Value{"cluster-id": cty.StringVal("123")}),
			shouldExist: false,
		},
		{
			name:         "when key for simple value provided than should return value",
			key:          "cluster-id",
			data:         cty.ObjectVal(map[string]cty.Value{"cluster-id": cty.StringVal("123")}),
			shouldExist:  true,
			expectedData: "123",
		},
		{
			name: "when target value is set to default value and includeZeroValues true, than should return exist flag true",
			key:  "foo.0.enabled",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(false),
								},
							),
						},
					),
				},
			),
			includeZeroValues: true,
			shouldExist:       true,
			expectedData:      false,
		},
		{
			name: "when target value is set to default value and includeZeroValues false, than should filter the default value and return exist flag false",
			key:  "foo.0.enabled",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(false),
								},
							),
						},
					),
				},
			),
			includeZeroValues: false,
			shouldExist:       false,
			expectedData:      nil,
		},
		{
			name: "when target value is set to default value and includeZeroValues false, than should filter the default value and return exist flag false",
			key:  "foo",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(false),
								},
							),
						},
					),
				},
			),
			includeZeroValues: false,
			shouldExist:       false,
			expectedData:      nil,
		},
		{
			name: "when given key is *schema.TypeList and unwrapSingleItemList is true, than should return a map instead of list",
			key:  "foo",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(true),
								},
							),
						},
					),
				},
			),
			unwrapSingleItemList: true,
			shouldExist:          true,
			expectedData:         map[string]any{"enabled": true},
		},
		{
			name: "when given key is *schema.TypeList and unwrapSingleItemList is false, than should return a list",
			key:  "foo",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(true),
								},
							),
						},
					),
				},
			),
			unwrapSingleItemList: false,
			includeZeroValues:    true,
			shouldExist:          true,
			expectedData:         []any{map[string]any{"enabled": true}},
		},
		{
			name: "when includeZeroValues true, than should also include properties set to zero values",
			key:  "foo",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(false),
									"bar": cty.ListVal(
										[]cty.Value{
											cty.ObjectVal(
												map[string]cty.Value{
													"baz": cty.NumberIntVal(0),
												},
											),
										},
									),
								},
							),
						},
					),
				},
			),
			unwrapSingleItemList: true,
			includeZeroValues:    true,
			shouldExist:          true,
			expectedData:         map[string]any{"enabled": false, "bar": map[string]any{"baz": 0}},
		},
		{
			name: "when includeZeroValues false, than should filter the zero values even if they are explicitly set",
			key:  "foo",
			data: cty.ObjectVal(
				map[string]cty.Value{
					"foo": cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(true),
									"bar": cty.ListVal(
										[]cty.Value{
											cty.ObjectVal(
												map[string]cty.Value{
													"baz": cty.NumberIntVal(0),
												},
											),
										},
									),
								},
							),
						},
					),
				},
			),
			unwrapSingleItemList: true,
			includeZeroValues:    false,
			shouldExist:          true,
			expectedData:         map[string]any{"enabled": true},
		},
	}

	for _, test := range tt {
		r := require.New(t)
		t.Run(test.name, func(t *testing.T) {
			state := terraform.NewInstanceStateShimmedFromValue(test.data, 0)
			resource := testResource()

			result, ok := extractNestedValues(resource.Data(state), test.key, test.unwrapSingleItemList, test.includeZeroValues)

			r.Equal(test.shouldExist, ok)
			r.Equal(test.expectedData, result)
		})
	}
}

func testResource() *schema.Resource {
	return &schema.Resource{
		Description: "Terraform resource for testing",
		Schema: map[string]*schema.Schema{
			"cluster-id": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id",
			},
			"foo": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "foo",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "enable/disable foo",
						},
						"bar": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "bar",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"baz": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: "count of baz",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
