package castai

import (
	"github.com/stretchr/testify/require"
	"testing"
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
