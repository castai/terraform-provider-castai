package castai

import (
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func convertStringArr(arr []interface{}) []string {
	var result []string
	for _, val := range arr {
		if val == nil {
			continue
		}
		result = append(result, val.(string))
	}
	return result
}

func toStringPtr(value string) *string {
	return &value
}

func toStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

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

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toStringSlice(arr *[]string) []string {
	if arr == nil {
		return nil
	}
	return *arr
}

func toInt32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func toBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func toBoolPtr(v bool) *bool {
	return &v
}

func toInt32Ptr(v int32) *int32 {
	return &v
}

func toCloudsStringSlice(clouds *[]sdk.CastaiV1Cloud) []string {
	if clouds == nil {
		return []string{}
	}
	out := make([]string, len(*clouds))
	for i, cloud := range *clouds {
		out[i] = string(cloud)
	}
	return out
}

func toCastaiClouds(clouds []interface{}) []sdk.CastaiV1Cloud {
	out := make([]sdk.CastaiV1Cloud, len(clouds))
	for _, cloud := range clouds {
		out = append(out, sdk.CastaiV1Cloud(cloud.(string)))
	}
	return out
}
