package castai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/types"
)

func toPtr[S any](src S) *S {
	return &src
}

func toStringMap(m map[string]interface{}) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.(string)
	}
	return out
}

func toStringList(items []interface{}) []string {
	out := make([]string, 0, len(items))
	for _, v := range items {
		val, ok := v.(string)
		if ok && val != "" {
			out = append(out, val)
		}
	}
	return out
}

func stringToMap(s string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	err := json.Unmarshal([]byte(s), &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toSection(d *schema.ResourceData, sectionName string) map[string]interface{} {
	val, ok := d.GetOk(sectionName)
	if !ok {
		return nil
	}
	sections := val.([]interface{})
	if len(sections) == 0 || sections[0] == nil {
		return nil
	}

	return sections[0].(map[string]interface{})
}

func toNestedSection(d *schema.ResourceData, parts ...string) map[string]interface{} {
	return toSection(d, strings.Join(parts, "."))
}

func readOptionalValue[T any](d map[string]any, key string) *T {
	val, ok := d[key]
	if !ok {
		return nil
	}

	return lo.ToPtr(val.(T))
}

func readOptionalValueOrDefault[T any](d map[string]any, key string, defaultValue T) T {
	val, ok := d[key]
	if !ok {
		return defaultValue
	}

	return val.(T)
}

func readAndConvertOptionalValue[Storage any, T any](d map[string]any, key string, conversion func(v Storage) T) *T {
	val := readOptionalValue[Storage](d, key)
	if val == nil {
		return nil
	}

	return lo.ToPtr(conversion(*val))
}

type Number interface {
	constraints.Integer | constraints.Float
}

func readOptionalNumber[Storage Number, T Number](d map[string]any, key string) *T {
	return readAndConvertOptionalValue[Storage, T](d, key, func(v Storage) T {
		return T(v)
	})
}

func readOptionalJson[T any](d map[string]any, key string) (*T, error) {
	val := readOptionalValue[string](d, key)
	if val == nil || *val == "" {
		return nil, nil
	}

	var out T
	err := json.Unmarshal([]byte(*val), &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func toNilList[T any](l *[]T) *[]T {
	if l == nil || len(*l) == 0 {
		return nil
	}
	return l
}

func normalizeJSON(bytes []byte) ([]byte, error) {
	var output interface{}
	err := json.Unmarshal(bytes, &output)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}

func getDefaultOrganizationId(ctx context.Context, meta any) (string, error) {
	response, err := meta.(*ProviderConfig).api.UsersAPIListOrganizationsWithResponse(ctx)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return "", fmt.Errorf("fetching organizations: %w", checkErr)
	}
	if len(response.JSON200.Organizations) == 0 {
		return "", fmt.Errorf("no organizations found")
	}

	// The first organization is the default one
	id := response.JSON200.Organizations[0].Id
	if id == nil {
		return "", fmt.Errorf("organization id is nil")
	}
	return *id, nil
}

// checkFloatAttr is a helper function for Terraform acceptance tests to check float attributes with a precision of 3
// decimal places. The attributes map is a map[string]string, so floats in there may be affected by the rounding errors.
func checkFloatAttr(resource, path string, val float64) func(state *terraform.State) error {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[resource]
		if !ok {
			return errors.New("resource not found")
		}
		v, ok := res.Primary.Attributes[path]
		if !ok {
			return errors.New("attribute not found")
		}
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		parsed = math.Round(parsed*1000) / 1000
		if parsed != val {
			return fmt.Errorf("expected %f, got %f", val, parsed)
		}
		return nil
	}
}

// extractNestedValues extracts nested values from a ResourceProvider (schema.ResourceData or schema.ResourceDiff)
// by their full key, checking if the value is set explicitly or to the default value.
//
// This function helps convert nested values to a map[string]interface{} for decoding to a struct using json.Unmarshal or mapstructure.Decode.
//
// Parameters:
//
// - provider, The types.ResourceProvider to extract values from.
//
// - key, The full key of the value to extract. Like "foo.bar.0.baz".
//
// - unwrapSingleItemList, If true, unwraps single item lists to the item itself. This is useful for complex types defined as
// *schema.TypeList with max item count 1.
//
// - includeDefaultValues, If true, includes values regardless of value being set explicitly or to the default value returned by the terraform provider.
func extractNestedValues(provider types.ResourceProvider, key string, unwrapSingleItemList bool, includeDefaultValues bool) (any, bool) {
	val, ok := provider.GetOk(key)

	isValExists := includeDefaultValues || ok
	if !isValExists {
		return nil, false
	}

	switch vt := val.(type) {
	case map[string]any:
		out := make(map[string]any)

		for nestedKey := range vt {
			if val, ok = extractNestedValues(provider, fmt.Sprintf("%s.%s", key, nestedKey), unwrapSingleItemList, includeDefaultValues); ok {
				out[nestedKey] = val
			}
		}

		if len(out) == 0 {
			return nil, false
		}

		return out, true
	case []any:
		if unwrapSingleItemList && len(vt) == 1 {
			if _, ok := vt[0].(map[string]any); ok {
				return extractNestedValues(provider, fmt.Sprintf("%s.%d", key, 0), unwrapSingleItemList, includeDefaultValues)
			}
		}

		var vals []any

		for i := range vt {
			nestedKey := fmt.Sprintf("%s.%d", key, i)

			if val, ok = extractNestedValues(provider, nestedKey, unwrapSingleItemList, includeDefaultValues); ok {
				vals = append(vals, val)
			}
		}

		if len(vals) == 0 {
			return nil, false
		}

		return vals, true
	}

	return val, isValExists
}
