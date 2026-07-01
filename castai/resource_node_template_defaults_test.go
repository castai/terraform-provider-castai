package castai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

// TestNodeTemplateMinCpuMemoryNilWhenZero asserts that toTemplateConstraints
// sends nil (which serializes to JSON null) for min_cpu and min_memory when
// they are not configured. This aligns min_* behavior with the pre-existing
// max_* behavior and prevents the provider from silently forcing "0" on the
// backend for fields the user never set.
func TestNodeTemplateMinCpuMemoryNilWhenZero(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]any
		wantMinCPU *int32
		wantMinMem *int32
	}{
		{
			name:       "unset - both fields become nil in request",
			input:      map[string]any{},
			wantMinCPU: nil,
			wantMinMem: nil,
		},
		{
			name: "explicit zero - both fields become nil in request",
			input: map[string]any{
				FieldNodeTemplateMinCpu:    0,
				FieldNodeTemplateMinMemory: 0,
			},
			wantMinCPU: nil,
			wantMinMem: nil,
		},
		{
			name: "non-zero values are sent verbatim",
			input: map[string]any{
				FieldNodeTemplateMinCpu:    4,
				FieldNodeTemplateMinMemory: 8192,
			},
			wantMinCPU: toPtr(int32(4)),
			wantMinMem: toPtr(int32(8192)),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			got := toTemplateConstraints(tc.input)
			r.NotNil(got)
			r.Equal(tc.wantMinCPU, got.MinCpu)
			r.Equal(tc.wantMinMem, got.MinMemory)
		})
	}
}

// TestNodeTemplateSpotInterruptionPredictionsTypeDefault asserts that when the
// user does not set spot_interruption_predictions_type, the schema default
// "interruption-predictions" is used and consequently sent to the backend.
// This matches the CAST AI UI default and avoids drift where the UI writes
// "interruption-predictions" but the provider would otherwise send "".
func TestNodeTemplateSpotInterruptionPredictionsTypeDefault(t *testing.T) {
	r := require.New(t)

	res := resourceNodeTemplate()
	sm := schema.InternalMap(res.Schema)

	config := terraform.NewResourceConfigRaw(map[string]any{
		FieldClusterId:             "b6bfc074-a267-400f-b8f1-db0850c369b1",
		FieldNodeTemplateName:      "default-by-castai",
		FieldNodeTemplateIsDefault: true,
		FieldNodeTemplateIsEnabled: true,
		FieldNodeTemplateConstraints: []any{
			map[string]any{
				FieldNodeTemplateOnDemand: true,
			},
		},
	})

	diff, err := sm.Diff(context.Background(), nil, config, nil, nil, true)
	r.NoError(err)

	d, err := sm.Data(nil, diff)
	r.NoError(err)

	constraintsList := d.Get(FieldNodeTemplateConstraints).([]any)
	r.Len(constraintsList, 1)
	constraints := constraintsList[0].(map[string]any)

	r.Equal("interruption-predictions", constraints[FieldNodeTemplateSpotInterruptionPredictionsType],
		"schema default should be interruption-predictions when the user omits spot_interruption_predictions_type")

	sent := toTemplateConstraints(constraints)
	r.NotNil(sent)
	r.NotNil(sent.SpotInterruptionPredictionsType)
	r.Equal("interruption-predictions", *sent.SpotInterruptionPredictionsType)
}

// TestNodeTemplateMinCpuMemoryJSONBody is an end-to-end guard that verifies
// the JSON payload sent to the CAST AI API contains "minCpu": null and
// "minMemory": null when the user has not configured those fields.
func TestNodeTemplateMinCpuMemoryJSONBody(t *testing.T) {
	r := require.New(t)

	res := resourceNodeTemplate()
	sm := schema.InternalMap(res.Schema)

	config := terraform.NewResourceConfigRaw(map[string]any{
		FieldClusterId:             "b6bfc074-a267-400f-b8f1-db0850c369b1",
		FieldNodeTemplateName:      "default-by-castai",
		FieldNodeTemplateIsDefault: true,
		FieldNodeTemplateIsEnabled: true,
		FieldNodeTemplateConstraints: []any{
			map[string]any{
				FieldNodeTemplateOnDemand: true,
			},
		},
	})

	diff, err := sm.Diff(context.Background(), nil, config, nil, nil, true)
	r.NoError(err)

	d, err := sm.Data(nil, diff)
	r.NoError(err)

	constraints := d.Get(FieldNodeTemplateConstraints).([]any)[0].(map[string]any)
	sent := toTemplateConstraints(constraints)

	body, err := json.Marshal(sent)
	r.NoError(err)

	bodyStr := string(body)
	r.Contains(bodyStr, `"minCpu":null`, "min_cpu should be sent as JSON null when not configured")
	r.Contains(bodyStr, `"minMemory":null`, "min_memory should be sent as JSON null when not configured")
	r.Contains(bodyStr, `"spotInterruptionPredictionsType":"interruption-predictions"`,
		"spot_interruption_predictions_type default should be interruption-predictions")
}
