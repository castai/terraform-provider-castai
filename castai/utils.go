package castai

import "encoding/json"

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
