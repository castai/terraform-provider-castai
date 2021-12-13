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
