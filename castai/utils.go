package castai

import (
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
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

func readOptionalValue[T any](d map[string]any, key string) *T {
	val, ok := d[key]
	if !ok {
		return nil
	}

	return lo.ToPtr(val.(T))
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
