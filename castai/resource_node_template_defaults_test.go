package castai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

// TestNodeTemplateMinCpuMemoryJSONBody verifies that
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
