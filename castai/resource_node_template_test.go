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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
				"name": "gpu",
				"constraints": {
				  "spot": false,
				  "onDemand": true,
				  "useSpotFallbacks": false,
				  "fallbackRestoreRateSeconds": 0,
				  "enableSpotDiversity": false,
				  "spotDiversityPriceIncreaseLimitPercent": 20,
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
					"excludeNames": []
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
				  ]
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
				"customInstancesWithExtendedMemoryEnabled": true
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
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "gpu"

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.ElementsMatch(strings.Split(`ID = gpu
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b1
configuration_id = 7dc4f922-29c9-4377-889c-0c8c5fb8d497
constraints.# = 1
constraints.0.architectures.# = 2
constraints.0.architectures.0 = amd64
constraints.0.architectures.1 = arm64
constraints.0.azs.# = 3
constraints.0.azs.0 = us-west-2a
constraints.0.azs.1 = us-west-2b
constraints.0.azs.2 = us-west-2c
constraints.0.compute_optimized = false
constraints.0.compute_optimized_state = disabled
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
constraints.0.spot = false
constraints.0.spot_diversity_price_increase_limit_percent = 20
constraints.0.spot_interruption_predictions_enabled = true
constraints.0.spot_interruption_predictions_type = aws-rebalance-recommendations
constraints.0.storage_optimized = false
constraints.0.storage_optimized_state = enabled
constraints.0.use_spot_fallbacks = false
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
`, "\n"),
		strings.Split(data.State().String(), "\n"),
	)
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

	resource := resourceNodeTemplate()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:        cty.StringVal(clusterId),
		FieldNodeTemplateName: cty.StringVal("gpu"),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "gpu"

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.NotNil(result)
	r.True(result.HasError())
	r.Equal(result[0].Summary, "failed to find node template with name: gpu")
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
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = "default-by-castai"

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
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
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

func TestAccResourceNodeTemplate_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-node-template-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_node_template.test"
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "cost-terraform")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckNodeTemplateDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccNodeTemplateConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "4"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "100"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.on_demand", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.0", "amd64"),
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
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.architectures.0", "arm64"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.os.0", "linux"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.0", "eu-central-1a"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.1", "eu-central-1b"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.azs.2", "eu-central-1c"),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.enable_spot_diversity", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot_diversity_price_increase_limit_percent", "22"),
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
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), testAccNodeConfig(rName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_clusterid.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = true

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
				}

				custom_priority {
					instance_families = ["c", "d"]
					spot = true
					on_demand = true
				}
			}
		}
	`, rName))
}

func testNodeTemplateUpdated(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), testAccNodeConfig(rName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_clusterid.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = true
			
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

			constraints {
				use_spot_fallbacks = true
				spot = true
				on_demand = true
				enable_spot_diversity = true
				spot_diversity_price_increase_limit_percent = 22
				spot_interruption_predictions_enabled = true
				spot_interruption_predictions_type = "interruption-predictions"
				fallback_restore_rate_seconds = 1800
				storage_optimized_state = "enabled"
				compute_optimized_state = "disabled"
				architectures = ["arm64"]
				burstable_instances = "enabled"
				customer_specific = "enabled"
				azs = ["eu-central-1a", "eu-central-1b", "eu-central-1c"]

				custom_priority {
					instance_families = ["a", "b"]
					spot = true
				}
				custom_priority {
					instance_families = ["c", "d"]
					spot = true
					on_demand = true
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
				return fmt.Errorf("node template %q still exists; %+v", id, *found.Template)
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
