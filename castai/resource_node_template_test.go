package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestNodeTemplateResourceReadContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	body := io.NopCloser(bytes.NewReader([]byte(`
		{
		  "items": [
			{
			  "template": {
				"configurationId": "7dc4f922-29c9-4377-889c-0c8c5fb8d497",
				"configurationName": "default",
				"isEnabled": true,
			   "gpu": {
				 "enableTimeSharing": true,
				 "defaultSharedClientsPerGpu": 10,
				 "userManagedGpuDrivers": true,
				 "sharingConfiguration": {
					"A100": {
					  "sharedClientsPerGpu": 5
					}
				 }
				},
				"name": "gpu",
				"constraints": {
				  "spot": false,
				  "onDemand": true,
				  "useSpotFallbacks": false,
				  "fallbackRestoreRateSeconds": 0,
				  "enableSpotDiversity": false,
				  "spotDiversityPriceIncreaseLimitPercent": 20,
				  "enableSpotReliability": true,
				  "spotReliabilityPriceIncreaseLimitPercent": 10,
				  "spotInterruptionPredictionsEnabled": true,
				  "spotInterruptionPredictionsType": "aws-rebalance-recommendations",
				  "storageOptimized": true,
				  "computeOptimized": false,
				  "minCpu": 10,
				  "maxCpu": 10000,
				  "instanceFamilies": {
					"include": [],
					"exclude": [
					  "p4d",
					  "p3dn",
					  "p2",
					  "g3s",
					  "g5g",
					  "g5",
					  "g3"
					]
				  },
	              "architectures": ["amd64", "arm64"],
				  "os": ["linux"],
				  "azs": ["us-west-2a", "us-west-2b", "us-west-2c"],
				  "gpu": {
					"manufacturers": [
					  "NVIDIA"
					],
					"includeNames": [],
					"excludeNames": [],
					"fractionalGPUs": "ENABLED"
				  },
				  "customPriority": [
				    {
						"families": ["a","b"],
				  		"spot": true,
				  		"onDemand": true
				  	}
				  ],
				  "dedicatedNodeAffinity": [
				    {	
						"name": "foo",
						"azName": "eu-central-1a",
						"instanceTypes": ["m5.24xlarge"],
						"affinity": [
                          {
							"key": "gke.io/gcp-nodepool",
							"operator": "In",	
							"values": ["foo"]
						  }
						]
					}
				  ],
				  "cpuManufacturers": ["INTEL", "AMD"],
	              "architecturePriority": ["amd64", "arm64"],
	              "resourceLimits": {
                    "cpuLimitEnabled": true,
                    "cpuLimitMaxCores": 20
                  }
				},
				"version": "3",
				"shouldTaint": true,
				"customLabels": {
					"key-1": "value-1",
					"key-2": "value-2"
				},
				"customTaints": [
				  {
				    "key": "some-key-1",
				    "value": "some-value-1",
				    "effect": "NoSchedule"
				  },
				  {
				    "key": "some-key-2",
				    "value": "some-value-2",
				    "effect": "NoSchedule"
				  }
				],
				"rebalancingConfig": {
				  "minNodes": 0
				},
				"customInstancesEnabled": true,
				"customInstancesWithExtendedMemoryEnabled": true,
				"edgeLocationIds": ["a1b2c3d4-e5f6-7890-abcd-ef1234567890", "b2c3d4e5-f6a7-8901-bcde-f12345678901"]
			  }
			}
		  ]
		}
	`)))
	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)}).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:        cty.StringVal(clusterId),
		FieldNodeTemplateName: cty.StringVal("gpu"),
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "gpu"

	data := resource.Data(state)
	// spew.Dump(data)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.ElementsMatch(strings.Split(`ID = gpu
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b1
configuration_id = 7dc4f922-29c9-4377-889c-0c8c5fb8d497
constraints.# = 1
constraints.0.architecture_priority.# = 2
constraints.0.architecture_priority.0 = amd64
constraints.0.architecture_priority.1 = arm64
constraints.0.architectures.# = 2
constraints.0.architectures.0 = amd64
constraints.0.architectures.1 = arm64
constraints.0.azs.# = 3
constraints.0.azs.0 = us-west-2a
constraints.0.azs.1 = us-west-2b
constraints.0.azs.2 = us-west-2c
constraints.0.compute_optimized = false
constraints.0.compute_optimized_state = disabled
constraints.0.cpu_manufacturers.# = 2
constraints.0.cpu_manufacturers.0 = INTEL
constraints.0.cpu_manufacturers.1 = AMD
constraints.0.custom_priority.# = 1
constraints.0.custom_priority.0.instance_families.# = 2
constraints.0.custom_priority.0.instance_families.0 = a
constraints.0.custom_priority.0.instance_families.1 = b
constraints.0.custom_priority.0.spot = true
constraints.0.custom_priority.0.on_demand = true
constraints.0.enable_spot_diversity = false
constraints.0.fallback_restore_rate_seconds = 0
constraints.0.gpu.# = 1
constraints.0.gpu.0.exclude_names.# = 0
constraints.0.gpu.0.include_names.# = 0
constraints.0.gpu.0.manufacturers.# = 1
constraints.0.gpu.0.manufacturers.0 = NVIDIA
constraints.0.gpu.0.max_count = 0
constraints.0.gpu.0.min_count = 0
constraints.0.gpu.0.fractional_gpus = enabled
constraints.0.burstable_instances = 
constraints.0.customer_specific = 
constraints.0.instance_families.# = 1
constraints.0.instance_families.0.exclude.# = 7
constraints.0.instance_families.0.exclude.0 = p4d
constraints.0.instance_families.0.exclude.1 = p3dn
constraints.0.instance_families.0.exclude.2 = p2
constraints.0.instance_families.0.exclude.3 = g3s
constraints.0.instance_families.0.exclude.4 = g5g
constraints.0.instance_families.0.exclude.5 = g5
constraints.0.instance_families.0.exclude.6 = g3
constraints.0.instance_families.0.include.# = 0
constraints.0.is_gpu_only = false
constraints.0.max_cpu = 10000
constraints.0.max_memory = 0
constraints.0.min_cpu = 10
constraints.0.min_memory = 0
constraints.0.dedicated_node_affinity.# = 1
constraints.0.dedicated_node_affinity.0.affinity.# = 1
constraints.0.dedicated_node_affinity.0.affinity.0.key = gke.io/gcp-nodepool
constraints.0.dedicated_node_affinity.0.affinity.0.operator = In
constraints.0.dedicated_node_affinity.0.affinity.0.values.# = 1
constraints.0.dedicated_node_affinity.0.affinity.0.values.0 = foo
constraints.0.dedicated_node_affinity.0.az_name = eu-central-1a
constraints.0.dedicated_node_affinity.0.instance_types.# = 1
constraints.0.dedicated_node_affinity.0.instance_types.0 = m5.24xlarge
constraints.0.dedicated_node_affinity.0.name = foo
constraints.0.on_demand = true
constraints.0.os.# = 1
constraints.0.os.0 = linux
constraints.0.resource_limits.# = 1
constraints.0.resource_limits.0.cpu_limit_enabled = true
constraints.0.resource_limits.0.cpu_limit_max_cores = 20
constraints.0.spot = false
constraints.0.spot_diversity_price_increase_limit_percent = 20
constraints.0.spot_reliability_enabled = true
constraints.0.spot_reliability_price_increase_limit_percent = 10
constraints.0.spot_interruption_predictions_enabled = true
constraints.0.spot_interruption_predictions_type = aws-rebalance-recommendations
constraints.0.storage_optimized = false
constraints.0.storage_optimized_state = enabled
constraints.0.use_spot_fallbacks = false
constraints.0.bare_metal = unspecified
custom_instances_enabled = true
custom_instances_with_extended_memory_enabled = true
custom_labels.% = 2
custom_labels.key-1 = value-1
custom_labels.key-2 = value-2
custom_taints.# = 2
custom_taints.0.effect = NoSchedule
custom_taints.0.key = some-key-1
custom_taints.0.value = some-value-1
custom_taints.1.effect = NoSchedule
custom_taints.1.key = some-key-2
custom_taints.1.value = some-value-2
is_default = false
is_enabled = true
name = gpu
rebalancing_config_min_nodes = 0
should_taint = true
Tainted = false
clm_enabled = false
edge_location_ids.# = 2
edge_location_ids.0 = a1b2c3d4-e5f6-7890-abcd-ef1234567890
edge_location_ids.1 = b2c3d4e5-f6a7-8901-bcde-f12345678901
gpu.# = 1
gpu.0.default_shared_clients_per_gpu = 10
gpu.0.enable_time_sharing = true
gpu.0.user_managed_gpu_drivers = true
gpu.0.sharing_configuration.# = 1
gpu.0.sharing_configuration.0.gpu_name = A100
gpu.0.sharing_configuration.0.shared_clients_per_gpu = 5
`, "\n"),
		strings.Split(data.State().String(), "\n"),
	)
}

func Test_flattenNodeAffinity(t *testing.T) {
	makeSDKNodeAffinityWithOperator := func(op string) []sdk.NodetemplatesV1TemplateConstraintsDedicatedNodeAffinity {
		return []sdk.NodetemplatesV1TemplateConstraintsDedicatedNodeAffinity{
			{
				Affinity: &[]sdk.K8sSelectorV1KubernetesNodeAffinity{{
					Key:      "kubernetes.io/os",
					Operator: sdk.K8sSelectorV1Operator(op),
					Values:   []string{"linux"},
				}},
				AzName:        lo.ToPtr("us-central1-c"),
				InstanceTypes: &[]string{"e2"},
				Name:          lo.ToPtr("linux-only"),
			},
		}
	}

	makeMappedNodeAffinityWithOperator := func(op string) []map[string]any {
		wantNA := []map[string]any{
			{
				FieldNodeTemplateInstanceTypes: []string{"e2"},
				FieldNodeTemplateAzName:        "us-central1-c",
				FieldNodeTemplateName:          "linux-only",
				FieldNodeTemplateAffinityName: []map[string]any{
					{
						FieldNodeTemplateAffinityKeyName:      "kubernetes.io/os",
						FieldNodeTemplateAffinityOperatorName: op,
						FieldNodeTemplateAffinityValuesName:   []string{"linux"},
					},
				},
			},
		}
		return wantNA
	}

	tt := []struct {
		name              string
		inputNodeAffinity []sdk.NodetemplatesV1TemplateConstraintsDedicatedNodeAffinity
		wantNodeAffinity  []map[string]any
		wantErr           bool
	}{
		{
			name:              "should produce an error for an unknown operator",
			inputNodeAffinity: makeSDKNodeAffinityWithOperator("UNKNOWN"),
			wantNodeAffinity:  makeMappedNodeAffinityWithOperator(""),
			wantErr:           true,
		},
	}

	for _, canonical := range nodeSelectorOperators {
		testedVariants := []string{canonical, strings.ToLower(canonical), strings.ToUpper(canonical)}
		for _, variant := range testedVariants {
			tcName := fmt.Sprintf("should map %q to %q", variant, canonical)
			input := makeSDKNodeAffinityWithOperator(variant)
			want := makeMappedNodeAffinityWithOperator(canonical)

			tc := struct {
				name              string
				inputNodeAffinity []sdk.NodetemplatesV1TemplateConstraintsDedicatedNodeAffinity
				wantNodeAffinity  []map[string]any
				wantErr           bool
			}{
				name:              tcName,
				inputNodeAffinity: input,
				wantNodeAffinity:  want,
				wantErr:           false,
			}

			tt = append(tt, tc)
		}
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			got, err := flattenNodeAffinity(tc.inputNodeAffinity)
			r.Equal(tc.wantNodeAffinity, got)
			if tc.wantErr {
				r.Error(err)
			}
		})
	}
}

func Test_flattenPriceAdjustmentConfiguration(t *testing.T) {
	tt := []struct {
		name  string
		input *sdk.NodetemplatesV1PriceAdjustmentConfiguration
		want  []map[string]any
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty configuration",
			input: &sdk.NodetemplatesV1PriceAdjustmentConfiguration{},
			want:  []map[string]any{{}},
		},
		{
			name: "configuration with adjustments",
			input: &sdk.NodetemplatesV1PriceAdjustmentConfiguration{
				InstanceTypeAdjustments: &map[string]string{
					"r7a.xlarge": "1.0",
					"r7i.xlarge": "1.20",
					"c6a.xlarge": "0.90",
				},
			},
			want: []map[string]any{
				{
					FieldNodeTemplateInstanceTypeAdjustments: map[string]string{
						"r7a.xlarge": "1.0",
						"r7i.xlarge": "1.20",
						"c6a.xlarge": "0.90",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			got := flattenPriceAdjustmentConfiguration(tc.input)
			r.Equal(tc.want, got)
		})
	}
}

func Test_toPriceAdjustmentConfiguration(t *testing.T) {
	tt := []struct {
		name  string
		input map[string]any
		want  *sdk.NodetemplatesV1PriceAdjustmentConfiguration
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty map returns empty configuration",
			input: map[string]any{},
			want:  &sdk.NodetemplatesV1PriceAdjustmentConfiguration{},
		},
		{
			name: "map with adjustments",
			input: map[string]any{
				FieldNodeTemplateInstanceTypeAdjustments: map[string]any{
					"r7a.xlarge": "1.0",
					"r7i.xlarge": "1.20",
				},
			},
			want: &sdk.NodetemplatesV1PriceAdjustmentConfiguration{
				InstanceTypeAdjustments: &map[string]string{
					"r7a.xlarge": "1.0",
					"r7i.xlarge": "1.20",
				},
			},
		},
		{
			name: "empty adjustments map",
			input: map[string]any{
				FieldNodeTemplateInstanceTypeAdjustments: map[string]any{},
			},
			want: &sdk.NodetemplatesV1PriceAdjustmentConfiguration{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			got := toPriceAdjustmentConfiguration(tc.input)
			r.Equal(tc.want, got)
		})
	}
}

func TestNodeTemplateResourceReadContextEmptyList(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	body := io.NopCloser(bytes.NewReader([]byte(`{"items": []}`)))
	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)}).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	nodeTemplate := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:        cty.StringVal(clusterId),
		FieldNodeTemplateName: cty.StringVal("gpu"),
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "gpu"

	data := nodeTemplate.Data(state)
	result := nodeTemplate.ReadContext(ctx, data, provider)

	r.Nil(result)
}

func TestNodeTemplateResourceCreate_defaultNodeTemplate(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	body := io.NopCloser(bytes.NewReader([]byte(`
		{
		  "items": [
			{
			  "template": {
				"configurationId": "7dc4f922-29c9-4377-889c-0c8c5fb8d497",
				"configurationName": "default",
				"name": "default-by-castai",
				"isEnabled": true,
				"isDefault": true,
				"clmEnabled": false,
				"constraints": {
				  "spot": false,
				  "onDemand": true,
				  "minCpu": 10,
				  "maxCpu": 10000,
				  "architectures": ["amd64", "arm64"],
	              "resourceLimits": {
                    "cpuLimitEnabled": true,
                    "cpuLimitMaxCores": 20
                  }
				},
				"version": "3",
				"shouldTaint": true,
				"customLabels": {},
				"customTaints": [],
				"rebalancingConfig": {
				  "minNodes": 0
				},
				"customInstancesEnabled": true,
				"customInstancesWithExtendedMemoryEnabled": true
			  }
			}
		  ]
		}
	`)))
	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)}).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	mockClient.EXPECT().
		NodeTemplatesAPIUpdateNodeTemplate(gomock.Any(), clusterId, "default-by-castai", gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte{}))}, nil)

	resource := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                                            cty.StringVal(clusterId),
		FieldNodeTemplateName:                                     cty.StringVal("default-by-castai"),
		FieldNodeTemplateIsDefault:                                cty.BoolVal(true),
		FieldNodeTemplateCustomInstancesEnabled:                   cty.BoolVal(true),
		FieldNodeTemplateCustomInstancesWithExtendedMemoryEnabled: cty.BoolVal(true),
		FieldNodeTemplateClmEnabled:                               cty.BoolVal(false),
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "default-by-castai"

	data := resource.Data(state)
	result := resource.CreateContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
}

func TestNodeTemplateResourceCreate_customNodeTemplate(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	name := "custom-template"
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	templateResponse := `
		{
		  "configurationId": "7dc4f922-29c9-4377-889c-0c8c5fb8d497",
		  "configurationName": "default",
		  "name": "custom-template",
		  "isEnabled": false,
		  "clmEnabled": true,
		  "constraints": {
		    "spot": false,
		    "onDemand": true,
		    "minCpu": 10,
		    "maxCpu": 10000,
		    "architectures": ["amd64", "arm64"],
		    "resourceLimits": {
		  	"cpuLimitEnabled": true,
		  	"cpuLimitMaxCores": 20
		    }
		  },
		  "version": "3",
		  "shouldTaint": true,
		  "customLabels": {},
		  "customTaints": [],
		  "rebalancingConfig": {
		    "minNodes": 0
		  },
		  "customInstancesEnabled": true,
		  "customInstancesWithExtendedMemoryEnabled": true
	    }
	`

	templateBody := io.NopCloser(bytes.NewReader([]byte(templateResponse)))
	listBody := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`
		{
		  "items": [
            {
              "template": %s
            }
		  ]
		}
	`, templateResponse))))

	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)}).
		Return(&http.Response{StatusCode: 200, Body: listBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
	mockClient.EXPECT().
		NodeTemplatesAPICreateNodeTemplate(gomock.Any(), clusterId, gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: templateBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                                            cty.StringVal(clusterId),
		FieldNodeTemplateName:                                     cty.StringVal(name),
		FieldNodeTemplateIsDefault:                                cty.BoolVal(true),
		FieldNodeTemplateIsEnabled:                                cty.BoolVal(false),
		FieldNodeTemplateCustomInstancesEnabled:                   cty.BoolVal(true),
		FieldNodeTemplateCustomInstancesWithExtendedMemoryEnabled: cty.BoolVal(true),
		FieldNodeTemplateClmEnabled:                               cty.BoolVal(true),
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = name

	data := resource.Data(state)
	result := resource.CreateContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
}

func TestNodeTemplateResourceDelete_defaultNodeTemplate(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	body := io.NopCloser(bytes.NewReader([]byte(`
		{
		  "items": [
			{
			  "template": {
				"configurationId": "7dc4f922-29c9-4377-889c-0c8c5fb8d497",
				"configurationName": "default",
				"name": "default-by-castai",
				"isEnabled": true,
				"isDefault": true,
				"constraints": {
				  "spot": false,
				  "onDemand": true,
				  "minCpu": 10,
				  "maxCpu": 10000,
				  "architectures": ["amd64", "arm64"]
				},
				"version": "3",
				"shouldTaint": true,
				"customLabels": {},
				"customTaints": [],
				"rebalancingConfig": {
				  "minNodes": 0
				},
				"customInstancesEnabled": true
			  }
			}
		  ]
		}
	`)))
	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)}).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:        cty.StringVal(clusterId),
		FieldNodeTemplateName: cty.StringVal("default-by-castai"),
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "default-by-castai"

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())

	result = resource.DeleteContext(ctx, data, provider)
	r.NotNil(result)
	r.Len(result, 1)
	r.False(result.HasError())
	r.Equal(diag.Warning, result[0].Severity)
	r.Equal("Skipping delete of \"default-by-castai\" node template", result[0].Summary)
	r.Equal("Default node templates cannot be deleted from CAST.ai. If you want to autoscaler to stop"+
		" considering this node template, you can disable it (either from UI or by setting `is_enabled` flag to"+
		" false).", result[0].Detail)
}

func TestAccEKS_ResourceNodeTemplate_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-node-template-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_node_template.test"
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "cost-terraform")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNodeTemplateDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccNodeTemplateConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
					resource.TestCheckResourceAttr(resourceName, "clm_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "custom_instances_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "custom_instances_with_extended_memory_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels."+rName+"-label-key-1", rName+"-label-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels."+rName+"-label-key-2", rName+"-label-value-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.key", rName+"-taint-key-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.value", rName+"-taint-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.key", rName+"-taint-key-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.value", rName+"-taint-value-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.effect", "NoExecute"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.2.key", rName+"-taint-key-3"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.2.value", rName+"-taint-value-3"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.2.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.3.key", rName+"-taint-key-4"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.3.value", ""),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.3.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.instance_families.0.exclude.0", "m5"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.manufacturers.0", "NVIDIA"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.include_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.exclude_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.fractional_gpus", "enabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "4"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "100"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.on_demand", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.0", "amd64"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architecture_priority.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architecture_priority.0", "amd64"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.0", "linux"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.0", "eu-central-1a"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.1", "eu-central-1b"),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.enable_spot_diversity", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_diversity_price_increase_limit_percent", "21"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_interruption_predictions_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_interruption_predictions_type", "interruption-predictions"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.0", "c"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.1", "d"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.on_demand", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.dedicated_node_affinity.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.storage_optimized_state", "disabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.compute_optimized_state", ""),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.0", "INTEL"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.1", "AMD"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.0.cpu_limit_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.0.cpu_limit_max_cores", "0"),
					resource.TestCheckResourceAttr(resourceName, "edge_location_ids.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_ids.1"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.m5.xlarge", "1.0"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.m5.2xlarge", "1.10"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.c5.xlarge", "0.95"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					clusterID := s.RootModule().Resources["castai_eks_cluster.test"].Primary.ID
					return fmt.Sprintf("%v/%v", clusterID, rName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testNodeTemplateUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
					resource.TestCheckResourceAttr(resourceName, "clm_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "custom_instances_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "custom_instances_with_extended_memory_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels."+rName+"-label-key-1", rName+"-label-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_labels."+rName+"-label-key-2", rName+"-label-value-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.key", rName+"-taint-key-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.value", rName+"-taint-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.key", rName+"-taint-key-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.value", ""),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.on_demand", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.instance_families.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.manufacturers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.include_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.exclude_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.fractional_gpus", "disabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.0", "arm64"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architecture_priority.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architecture_priority.0", "arm64"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.0", "linux"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.0", "eu-central-1a"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.1", "eu-central-1b"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.2", "eu-central-1c"),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.enable_spot_diversity", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_diversity_price_increase_limit_percent", "22"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_reliability_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_reliability_price_increase_limit_percent", "15"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_interruption_predictions_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_interruption_predictions_type", "interruption-predictions"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.0", "a"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.instance_families.1", "b"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.0.on_demand", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.1.instance_families.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.1.instance_families.0", "c"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.1.instance_families.1", "d"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.1.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.custom_priority.1.on_demand", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.dedicated_node_affinity.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.storage_optimized_state", "enabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.compute_optimized_state", "disabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.burstable_instances", "enabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.customer_specific", "enabled"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.0", "INTEL"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.cpu_manufacturers.1", "AMD"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.0.cpu_limit_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.resource_limits.0.cpu_limit_max_cores", "50"),
					resource.TestCheckResourceAttr(resourceName, "gpu.0.default_shared_clients_per_gpu", "1"),
					resource.TestCheckResourceAttr(resourceName, "gpu.0.enable_time_sharing", "false"),
					resource.TestCheckResourceAttr(resourceName, "edge_location_ids.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.m5.xlarge", "1.05"),
					resource.TestCheckResourceAttr(resourceName, "price_adjustment_configuration.0.instance_type_adjustments.r5.xlarge", "0.90"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 4.0",
			},
		},
	})
}

func testAccNodeTemplateConfig(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), testAccNodeConfig(rName), testAccEdgeLocationsConfig(rName, clusterName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_cluster.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = true
			clm_enabled = false

			custom_labels = {
				%[1]s-label-key-1 = "%[1]s-label-value-1"
				%[1]s-label-key-2 = "%[1]s-label-value-2"
			}

			custom_taints {
				key = "%[1]s-taint-key-1"
				value = "%[1]s-taint-value-1"
				effect = "NoSchedule"
			}

			custom_taints {
				key = "%[1]s-taint-key-2"
				value = "%[1]s-taint-value-2"
				effect = "NoExecute"
			}

			custom_taints {
				key = "%[1]s-taint-key-3"
				value = "%[1]s-taint-value-3"
			}

			custom_taints {
				key = "%[1]s-taint-key-4"
			}

			edge_location_ids = [castai_edge_location.test_1.id, castai_edge_location.test_2.id]

			price_adjustment_configuration {
				instance_type_adjustments = {
					"m5.xlarge"  = "1.0"
					"m5.2xlarge" = "1.10"
					"c5.xlarge"  = "0.95"
				}
			}

			constraints {
				fallback_restore_rate_seconds = 1800
				spot = true
				enable_spot_diversity = true
				spot_diversity_price_increase_limit_percent = 21
				spot_interruption_predictions_enabled = true
				spot_interruption_predictions_type = "interruption-predictions"
				use_spot_fallbacks = true
				storage_optimized_state = "disabled"
				burstable_instances = "enabled"
				customer_specific = "enabled"
				min_cpu = 4
				max_cpu = 100
				instance_families {
				  exclude = ["m5"]
				}
				azs = ["eu-central-1a", "eu-central-1b"]
				gpu {
					include_names = []
					exclude_names = []
					manufacturers = ["NVIDIA"]
					fractional_gpus = "enabled"
				}

				custom_priority {
					instance_families = ["c", "d"]
					spot = true
					on_demand = true
				}

				resource_limits {
					cpu_limit_enabled = false
					cpu_limit_max_cores = 0
				}

				cpu_manufacturers = ["INTEL", "AMD"]
				architecture_priority = ["amd64"]
			}
		}
	`, rName))
}

func testAccEdgeLocationsConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()
	return fmt.Sprintf(`
resource "castai_omni_cluster" "test_omni" {
  organization_id = %[1]q
  cluster_id      = castai_eks_cluster.test.id
}

resource "castai_edge_location" "test_1" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test_omni.id
  name            = "edge-loc-1"
  description     = "Test edge location 1"
  region          = "us-east-1"
  zones = [
    {
      id   = "us-east-1a"
      name = "us-east-1a"
    },
    {
      id   = "us-east-1b"
      name = "us-east-1b"
    }
  ]

  aws = {
	# fake credentials for testing purposes only
    account_id           = "123456789012"
    access_key_id_wo     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    vpc_id               = "vpc-12345678"
    security_group_id    = "sg-12345678"
	vpc_peered           = false
    subnet_ids = {
      "us-east-1a" = "subnet-12345678"
      "us-east-1b" = "subnet-12345679"
    }
    name_tag = "test-edge-location-1"
  }
}

resource "castai_edge_location" "test_2" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test_omni.id
  name            = "edge-loc-2"
  description     = "Test edge location 2"
  region          = "us-west-2"
  zones = [
    {
      id   = "us-west-2a"
      name = "us-west-2a"
    }
  ]

  aws = {
	# fake credentials for testing purposes only
    account_id           = "123456789012"
    access_key_id_wo     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    vpc_id               = "vpc-87654321"
    security_group_id    = "sg-87654321"
	vpc_peered           = false
    subnet_ids = {
      "us-west-2a" = "subnet-87654321"
    }
    name_tag = "test-edge-location-2"
  }
}
`, organizationID)
}

func testNodeTemplateUpdated(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), testAccNodeConfig(rName), testAccEdgeLocationsConfig(rName, clusterName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_cluster.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = true
			clm_enabled = true
			
			custom_labels = {
				%[1]s-label-key-1 = "%[1]s-label-value-1"
				%[1]s-label-key-2 = "%[1]s-label-value-2"
			}

			custom_taints {
				key = "%[1]s-taint-key-1"
				value = "%[1]s-taint-value-1"
				effect = "NoSchedule"
			}

			custom_taints {
				key = "%[1]s-taint-key-2"
				effect = "NoSchedule"
			}

			gpu {
			  default_shared_clients_per_gpu = 1
			  enable_time_sharing            = false
			}

			edge_location_ids = [castai_edge_location.test_2.id]

			price_adjustment_configuration {
				instance_type_adjustments = {
					"m5.xlarge"   = "1.05"
					"r5.xlarge"   = "0.90"
				}
			}

			constraints {
				use_spot_fallbacks = true
				spot = true
				on_demand = true
				enable_spot_diversity = false
				spot_diversity_price_increase_limit_percent = 22
				spot_reliability_enabled = true
				spot_reliability_price_increase_limit_percent = 15
				spot_interruption_predictions_enabled = true
				spot_interruption_predictions_type = "interruption-predictions"
				fallback_restore_rate_seconds = 1800
				storage_optimized_state = "enabled"
				compute_optimized_state = "disabled"
				architectures = ["arm64"]
				burstable_instances = "enabled"
				customer_specific = "enabled"
				azs = ["eu-central-1a", "eu-central-1b", "eu-central-1c"]
				bare_metal = false

				custom_priority {
					instance_families = ["a", "b"]
					spot = true
				}
				custom_priority {
					instance_families = ["c", "d"]
					spot = true
					on_demand = true
				}

				cpu_manufacturers = ["INTEL", "AMD"]
				architecture_priority = ["arm64"]

				gpu {
					fractional_gpus = "disabled"
				}

				resource_limits {
					cpu_limit_enabled   = true
					cpu_limit_max_cores = 50
				}
			}
		}
	`, rName))
}

func testAccCheckNodeTemplateDestroy(templateName string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client := testAccProvider.Meta().(*ProviderConfig).api
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "castai_node_template" {
				continue
			}

			id := rs.Primary.ID
			clusterID := rs.Primary.Attributes["cluster_id"]
			response, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(false)})
			if err != nil {
				return err
			}
			if response.StatusCode() == http.StatusNotFound {
				return nil
			}
			if found, ok := lo.Find(*response.JSON200.Items, func(item sdk.NodetemplatesV1NodeTemplateListItem) bool {
				return lo.FromPtr(item.Template.Name) == templateName
			}); ok {
				return fmt.Errorf("node template %q still exists; %+v", id, found.Template)
			}
			return nil
		}

		return nil
	}
}

func testAccNodeConfig(rName string) string {
	return ConfigCompose(fmt.Sprintf(`
data "aws_subnets" "cost" {
	tags = {
		Name = "*cost-terraform-cluster/SubnetPublic*"
	}
}

resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_eks_cluster.test.id
  disk_cpu_ratio    = 35
  subnets   	    = data.aws_subnets.cost.ids
  container_runtime = "dockerd"
  tags = {
    env = "development"
  }
  eks {
	instance_profile_arn = aws_iam_instance_profile.test.arn
    dns_cluster_ip       = "10.100.0.10"
	security_groups      = [aws_security_group.test.id]
  }
}

resource "castai_node_configuration_default" "default" {
  cluster_id        = castai_eks_cluster.test.id
  configuration_id  = castai_node_configuration.test.id
}

`, rName))
}

func Test_toTemplateGpu(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		want  *sdk.NodetemplatesV1GPU
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty map returns nil",
			input: map[string]any{},
			want:  nil,
		},
		{
			name: "user_managed_gpu_drivers true",
			input: map[string]any{
				FieldNodeTemplateUserManagedGPUDrivers: true,
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: nil, // Not set when 0 to avoid API validation error
				EnableTimeSharing:          lo.ToPtr(false),
				SharingConfiguration:       &map[string]sdk.NodetemplatesV1SharedGPU{},
				UserManagedGpuDrivers:      lo.ToPtr(true),
			},
		},
		{
			name: "user_managed_gpu_drivers false",
			input: map[string]any{
				FieldNodeTemplateUserManagedGPUDrivers: false,
			},
			want: nil, // All fields are zero/false, so returns nil
		},
		{
			name: "user_managed_gpu_drivers with time sharing enabled",
			input: map[string]any{
				FieldNodeTemplateUserManagedGPUDrivers:      true,
				FieldNodeTemplateEnableTimeSharing:          true,
				FieldNodeTemplateDefaultSharedClientsPerGpu: 8,
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(8)),
				EnableTimeSharing:          lo.ToPtr(true),
				SharingConfiguration:       &map[string]sdk.NodetemplatesV1SharedGPU{},
				UserManagedGpuDrivers:      lo.ToPtr(true),
			},
		},
		{
			name: "user_managed_gpu_drivers with sharing configuration",
			input: map[string]any{
				FieldNodeTemplateUserManagedGPUDrivers:      true,
				FieldNodeTemplateEnableTimeSharing:          true,
				FieldNodeTemplateDefaultSharedClientsPerGpu: 10,
				FieldNodeTemplateSharingConfiguration: []any{
					map[string]any{
						FieldNodeTemplateSharedGpuName:       "A100",
						FieldNodeTemplateSharedClientsPerGpu: 5,
					},
				},
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(10)),
				EnableTimeSharing:          lo.ToPtr(true),
				SharingConfiguration: &map[string]sdk.NodetemplatesV1SharedGPU{
					"A100": {
						SharedClientsPerGpu: lo.ToPtr(int32(5)),
					},
				},
				UserManagedGpuDrivers: lo.ToPtr(true),
			},
		},
		{
			name: "all gpu settings enabled",
			input: map[string]any{
				FieldNodeTemplateUserManagedGPUDrivers:      true,
				FieldNodeTemplateEnableTimeSharing:          true,
				FieldNodeTemplateDefaultSharedClientsPerGpu: 12,
				FieldNodeTemplateSharingConfiguration: []any{
					map[string]any{
						FieldNodeTemplateSharedGpuName:       "V100",
						FieldNodeTemplateSharedClientsPerGpu: 4,
					},
					map[string]any{
						FieldNodeTemplateSharedGpuName:       "T4",
						FieldNodeTemplateSharedClientsPerGpu: 8,
					},
				},
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(12)),
				EnableTimeSharing:          lo.ToPtr(true),
				SharingConfiguration: &map[string]sdk.NodetemplatesV1SharedGPU{
					"V100": {
						SharedClientsPerGpu: lo.ToPtr(int32(4)),
					},
					"T4": {
						SharedClientsPerGpu: lo.ToPtr(int32(8)),
					},
				},
				UserManagedGpuDrivers: lo.ToPtr(true),
			},
		},
		{
			name: "only time sharing without user managed drivers",
			input: map[string]any{
				FieldNodeTemplateEnableTimeSharing:          true,
				FieldNodeTemplateDefaultSharedClientsPerGpu: 6,
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(6)), // Set when > 0
				EnableTimeSharing:          lo.ToPtr(true),
				SharingConfiguration:       &map[string]sdk.NodetemplatesV1SharedGPU{},
				// UserManagedGpuDrivers not set (nil) - only sent when explicitly true (GKE-only)
			},
		},
		{
			name: "only enable_time_sharing true without default clients",
			input: map[string]any{
				FieldNodeTemplateEnableTimeSharing: true,
			},
			want: &sdk.NodetemplatesV1GPU{
				DefaultSharedClientsPerGpu: nil, // Not set when 0
				EnableTimeSharing:          lo.ToPtr(true),
				SharingConfiguration:       &map[string]sdk.NodetemplatesV1SharedGPU{},
				// UserManagedGpuDrivers not set (nil) - only sent when explicitly true (GKE-only)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toTemplateGpu(tt.input)

			if tt.want == nil {
				require.Nil(t, got)
				return
			}

			require.NotNil(t, got)

			// Compare DefaultSharedClientsPerGpu (can be nil)
			if tt.want.DefaultSharedClientsPerGpu == nil {
				require.Nil(t, got.DefaultSharedClientsPerGpu, "DefaultSharedClientsPerGpu should be nil")
			} else {
				require.NotNil(t, got.DefaultSharedClientsPerGpu, "DefaultSharedClientsPerGpu should not be nil")
				require.Equal(t, *tt.want.DefaultSharedClientsPerGpu, *got.DefaultSharedClientsPerGpu)
			}

			require.Equal(t, tt.want.EnableTimeSharing, got.EnableTimeSharing)
			require.Equal(t, tt.want.UserManagedGpuDrivers, got.UserManagedGpuDrivers)

			if tt.want.SharingConfiguration != nil && got.SharingConfiguration != nil {
				wantMap := *tt.want.SharingConfiguration
				gotMap := *got.SharingConfiguration

				// Both nil maps and empty maps are acceptable as equivalent
				if (wantMap == nil && gotMap == nil) || (len(wantMap) == 0 && len(gotMap) == 0) {
					// Pass - both are empty/nil
				} else {
					require.Equal(t, wantMap, gotMap)
				}
			} else {
				// Both should be nil
				require.Equal(t, tt.want.SharingConfiguration, got.SharingConfiguration, "SharingConfiguration pointers should be equal")
			}
		})
	}
}

func Test_flattenGpuSettings(t *testing.T) {
	tests := []struct {
		name    string
		input   *sdk.NodetemplatesV1GPU
		want    []map[string]any
		wantErr bool
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:  "empty gpu settings",
			input: &sdk.NodetemplatesV1GPU{},
			want: []map[string]any{
				{},
			},
			wantErr: false,
		},
		{
			name: "user_managed_gpu_drivers true",
			input: &sdk.NodetemplatesV1GPU{
				UserManagedGpuDrivers: lo.ToPtr(true),
			},
			want: []map[string]any{
				{
					FieldNodeTemplateUserManagedGPUDrivers: lo.ToPtr(true),
				},
			},
			wantErr: false,
		},
		{
			name: "user_managed_gpu_drivers false",
			input: &sdk.NodetemplatesV1GPU{
				UserManagedGpuDrivers: lo.ToPtr(false),
			},
			want: []map[string]any{
				{
					FieldNodeTemplateUserManagedGPUDrivers: lo.ToPtr(false),
				},
			},
			wantErr: false,
		},
		{
			name: "user_managed_gpu_drivers with time sharing",
			input: &sdk.NodetemplatesV1GPU{
				UserManagedGpuDrivers:      lo.ToPtr(true),
				EnableTimeSharing:          lo.ToPtr(true),
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(10)),
			},
			want: []map[string]any{
				{
					FieldNodeTemplateUserManagedGPUDrivers:      lo.ToPtr(true),
					FieldNodeTemplateEnableTimeSharing:          lo.ToPtr(true),
					FieldNodeTemplateDefaultSharedClientsPerGpu: lo.ToPtr(int32(10)),
				},
			},
			wantErr: false,
		},
		{
			name: "all gpu settings",
			input: &sdk.NodetemplatesV1GPU{
				UserManagedGpuDrivers:      lo.ToPtr(true),
				EnableTimeSharing:          lo.ToPtr(true),
				DefaultSharedClientsPerGpu: lo.ToPtr(int32(12)),
				SharingConfiguration: &map[string]sdk.NodetemplatesV1SharedGPU{
					"A100": {
						SharedClientsPerGpu: lo.ToPtr(int32(5)),
					},
					"V100": {
						SharedClientsPerGpu: lo.ToPtr(int32(3)),
					},
				},
			},
			want: []map[string]any{
				{
					FieldNodeTemplateUserManagedGPUDrivers:      lo.ToPtr(true),
					FieldNodeTemplateEnableTimeSharing:          lo.ToPtr(true),
					FieldNodeTemplateDefaultSharedClientsPerGpu: lo.ToPtr(int32(12)),
					FieldNodeTemplateSharingConfiguration: []map[string]any{
						{
							FieldNodeTemplateSharedGpuName:       "A100",
							FieldNodeTemplateSharedClientsPerGpu: int32(5),
						},
						{
							FieldNodeTemplateSharedGpuName:       "V100",
							FieldNodeTemplateSharedClientsPerGpu: int32(3),
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flattenGpuSettings(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.want))

			if len(tt.want) == 0 {
				return
			}

			// Compare all fields except sharing_configuration
			require.Equal(t, tt.want[0][FieldNodeTemplateUserManagedGPUDrivers], got[0][FieldNodeTemplateUserManagedGPUDrivers])
			require.Equal(t, tt.want[0][FieldNodeTemplateEnableTimeSharing], got[0][FieldNodeTemplateEnableTimeSharing])
			require.Equal(t, tt.want[0][FieldNodeTemplateDefaultSharedClientsPerGpu], got[0][FieldNodeTemplateDefaultSharedClientsPerGpu])

			// Compare sharing_configuration with ElementsMatch (order doesn't matter)
			wantSharing := tt.want[0][FieldNodeTemplateSharingConfiguration]
			gotSharing := got[0][FieldNodeTemplateSharingConfiguration]
			if wantSharing != nil || gotSharing != nil {
				require.ElementsMatch(t, wantSharing, gotSharing)
			}
		})
	}
}
