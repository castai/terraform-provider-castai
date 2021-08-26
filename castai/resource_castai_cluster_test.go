package castai

import (
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"reflect"
	"testing"
)

func TestExpandAutoscalerPolicies(t *testing.T) {
	expanded := []map[string]interface{}{
		{
			PolicyFieldEnabled: true,
			PolicyFieldClusterLimits: []map[string]interface{}{
				{
					PolicyFieldEnabled: false,
					PolicyFieldClusterLimitsCPU: []map[string]interface{}{
						{
							PolicyFieldClusterLimitsCPUmax: 100,
							PolicyFieldClusterLimitsCPUmin: 1,
						},
					},
				},
			},
			PolicyFieldNodeDownscaler: []map[string]interface{}{
				{
					PolicyFieldNodeDownscalerEmptyNodes: []map[string]interface{}{
						{
							PolicyFieldEnabled:                       false,
							PolicyFieldNodeDownscalerEmptyNodesDelay: 60,
						},
					},
				},
			},
			PolicyFieldSpotInstances: []map[string]interface{}{
				{
					PolicyFieldEnabled: false,
					//PolicyFieldSpotInstancesClouds: ["gcp"],
				},
			},
			PolicyFieldUnschedulablePods: []map[string]interface{}{
				{
					PolicyFieldEnabled: false,
					PolicyFieldUnschedulablePodsHeadroom: []map[string]interface{}{
						{
							PolicyFieldEnabled:                       false,
							PolicyFieldUnschedulablePodsHeadroomCPUp: 10,
							PolicyFieldUnschedulablePodsHeadroomRAMp: 10,
						},
					},
					PolicyFieldUnschedulablePodsNodeConstraint: []map[string]interface{}{
						{
							PolicyFieldEnabled: false,
							PolicyFieldUnschedulablePodsNodeConstraintMaxCPU: 32,
							PolicyFieldUnschedulablePodsNodeConstraintMaxRAM: 65536 / 1024.0,
							PolicyFieldUnschedulablePodsNodeConstraintMinCPU: 8,
							PolicyFieldUnschedulablePodsNodeConstraintMinRAM: 16384 / 1024.0,
						},
					},
				},
			},
		},
	}

	//policies :=
	expandAutoscalerPolicies(expanded[0])

	expected := sdk.UpsertPoliciesJSONRequestBody{
		ClusterLimits:     sdk.ClusterLimitsPolicy{},
		Enabled:           true,
		NodeDownscaler:    &sdk.NodeDownscaler{},
		SpotInstances:     sdk.SpotInstances{},
		UnschedulablePods: sdk.UnschedulablePodsPolicy{},
	}

	if !reflect.DeepEqual(expanded[0], expected) {
		t.Fatalf("bad test output: \n\n%#v\n\nExpected:\n\n%#v\n",
			expanded[0], expected)
	}
}
