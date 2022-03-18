package castai

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

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toInt32Ptr(v int32) *int32 {
	return &v
}

func toStringSlice(arr *[]string) []string {
	if arr == nil {
		return nil
	}
	return *arr
}

func toBoolPtr(v bool) *bool {
	return &v
}
