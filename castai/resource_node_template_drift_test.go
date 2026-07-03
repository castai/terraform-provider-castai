package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

// TestNodeTemplateSpotFieldsNoDriftWhenUnset reproduces CSU-5463. When the API
// returns spot-related constraint values but the user's Terraform configuration
// does not set them, the provider should not produce a perpetual diff. The fix
// is to mark those fields as Computed so that Terraform uses the API/state
// value instead of the zero value when the field is absent from configuration.
func TestNodeTemplateSpotFieldsNoDriftWhenUnset(t *testing.T) {
	r := require.New(t)
	res := resourceNodeTemplate()
	sm := schema.InternalMap(res.Schema)

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	name := "default-by-castai"

	// State returned by the API / stored in Terraform state.
	state := &terraform.InstanceState{
		ID: name,
		Attributes: map[string]string{
			FieldClusterId:                                             clusterID,
			FieldNodeTemplateName:                                      name,
			fmt.Sprintf("%s.#", FieldNodeTemplateConstraints):          "1",
			fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateOnDemand): "true",
			fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateMinCpu):   "2",
			fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent):   "21",
			fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotReliabilityPriceIncreaseLimitPercent): "20",
			fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotInterruptionPredictionsType):          "interruption-predictions",
		},
	}

	// User configuration that does NOT set the spot-related fields.
	config := terraform.NewResourceConfigRaw(map[string]interface{}{
		FieldClusterId:        clusterID,
		FieldNodeTemplateName: name,
		FieldNodeTemplateConstraints: []interface{}{
			map[string]interface{}{
				FieldNodeTemplateOnDemand: true,
				FieldNodeTemplateMinCpu:   2,
			},
		},
	})

	diff, err := sm.Diff(context.Background(), state, config, nil, nil, true)
	r.NoError(err)

	d, err := sm.Data(state, diff)
	r.NoError(err)

	spotDiversityPath := fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent)
	spotReliabilityPath := fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotReliabilityPriceIncreaseLimitPercent)
	spotTypePath := fmt.Sprintf("%s.0.%s", FieldNodeTemplateConstraints, FieldNodeTemplateSpotInterruptionPredictionsType)

	fmt.Printf("d.Get(%q) = %#v\n", spotDiversityPath, d.Get(spotDiversityPath))
	fmt.Printf("d.Get(%q) = %#v\n", spotReliabilityPath, d.Get(spotReliabilityPath))
	fmt.Printf("d.Get(%q) = %#v\n", spotTypePath, d.Get(spotTypePath))
	fmt.Printf("d.HasChange(%q) = %v\n", spotDiversityPath, d.HasChange(spotDiversityPath))
	fmt.Printf("d.HasChange(%q) = %v\n", spotReliabilityPath, d.HasChange(spotReliabilityPath))
	fmt.Printf("d.HasChange(%q) = %v\n", spotTypePath, d.HasChange(spotTypePath))

	// Simulate what the provider would send to the API on update.
	// updateNodeTemplate does: req.Constraints = toTemplateConstraints(v[0].(map[string]any))
	constraintsConfig := d.Get(FieldNodeTemplateConstraints).([]interface{})[0].(map[string]interface{})
	sentConstraints := toTemplateConstraints(constraintsConfig)
	sentJSON, err := json.MarshalIndent(sentConstraints, "", "  ")
	r.NoError(err)
	fmt.Printf("Constraints JSON sent to API on update:\n%s\n", string(sentJSON))

	// With the fix these fields are Computed; when absent from config Terraform
	// should keep the state value and report no change.
	r.False(d.HasChange(spotDiversityPath), "spot_diversity_price_increase_limit_percent should not drift when not configured")
	r.False(d.HasChange(spotReliabilityPath), "spot_reliability_price_increase_limit_percent should not drift when not configured")
	r.False(d.HasChange(spotTypePath), "spot_interruption_predictions_type should not drift when not configured")
}
