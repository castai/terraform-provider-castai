package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
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
				"name": "gpu",
				"constraints": {
				  "spot": false,
				  "useSpotFallbacks": false,
				  "fallbackRestoreRateSeconds": 0,
				  "storageOptimized": false,
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
				  "gpu": {
					"manufacturers": [
					  "NVIDIA"
					],
					"includeNames": [],
					"excludeNames": []
				  }
				},
				"version": "3",
				"shouldTaint": true,
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
				}
			  }
			}
		  ]
		}
	`)))
	mockClient.EXPECT().
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId).
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
	r.Equal(`ID = gpu
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b1
configuration_id = 7dc4f922-29c9-4377-889c-0c8c5fb8d497
constraints.# = 1
constraints.0.compute_optimized = false
constraints.0.fallback_restore_rate_seconds = 0
constraints.0.gpu.# = 1
constraints.0.gpu.0.exclude_names.# = 0
constraints.0.gpu.0.include_names.# = 0
constraints.0.gpu.0.manufacturers.# = 1
constraints.0.gpu.0.manufacturers.0 = NVIDIA
constraints.0.gpu.0.max_count = 0
constraints.0.gpu.0.min_count = 0
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
constraints.0.max_cpu = 10000
constraints.0.max_memory = 0
constraints.0.min_cpu = 10
constraints.0.min_memory = 0
constraints.0.spot = false
constraints.0.storage_optimized = false
constraints.0.use_spot_fallbacks = false
custom_label.# = 0
custom_taints.# = 2
custom_taints.0.effect = NoSchedule
custom_taints.0.key = some-key-1
custom_taints.0.value = some-value-1
custom_taints.1.effect = NoSchedule
custom_taints.1.key = some-key-2
custom_taints.1.value = some-value-2
name = gpu
rebalancing_config_min_nodes = 0
should_taint = true
Tainted = false
`, data.State().String())
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
		NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId).
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

func TestAccResourceNodeTemplate_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-node-template-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_node_template.test"
	clusterName := "cost-terraform"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckNodeTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNodeTemplateConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
					resource.TestCheckResourceAttr(resourceName, "custom_label.0.key", "custom-key-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_label.0.value", "custom-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.key", "custom-taint-key-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.value", "custom-taint-value-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.key", "custom-taint-key-2"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.1.value", "custom-taint-value-2"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.instance_families.0.exclude.0", "m5"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.manufacturers.0", "NVIDIA"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.include_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.exclude_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "4"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "100"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
					resource.TestCheckResourceAttr(resourceName, "custom_label.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.key", "custom-taint-key-1"),
					resource.TestCheckResourceAttr(resourceName, "custom_taints.0.value", "custom-taint-value-1"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.instance_families.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.manufacturers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.include_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.gpu.0.exclude_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.min_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.max_cpu", "0"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.use_spot_fallbacks", "true"),
					resource.TestCheckResourceAttr(resourceName, "constraints.0.spot", "true"),
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

			custom_label {
				key = "custom-key-1"
				value = "custom-value-1"
			}

			custom_taints {
				key = "custom-taint-key-1"
				value = "custom-taint-value-1"
				effect = "NoSchedule"
			}

			custom_taints {
				key = "custom-taint-key-2"
				value = "custom-taint-value-2"
				effect = "NoSchedule"
			}

			constraints {
				fallback_restore_rate_seconds = 1800
				spot = true
				use_spot_fallbacks = true
				min_cpu = 4
				max_cpu = 100
				instance_families {
				  exclude = ["m5"]
				}
				gpu {
					include_names = []
					exclude_names = []
					manufacturers = ["NVIDIA"]
				}	
				compute_optimized = false
				storage_optimized = false
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

			custom_taints {
				key = "custom-taint-key-1"
				value = "custom-taint-value-1"
				effect = "NoSchedule"
			}

			constraints {
				use_spot_fallbacks = true
				spot = true 
				fallback_restore_rate_seconds = 1800
				storage_optimized = false
				compute_optimized = false
			}
		}
	`, rName))
}

func testAccCheckNodeTemplateDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_node_template" {
			continue
		}

		id := rs.Primary.ID
		clusterID := rs.Primary.Attributes["cluster_id"]
		response, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID)
		if err != nil {
			return err
		}
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}
		if len(*response.JSON200.Items) == 0 {
			// Should be no templates
			return nil
		}

		return fmt.Errorf("node template %q still exists", id)
	}

	return nil
}

func testAccNodeConfig(rName string) string {
	return ConfigCompose(fmt.Sprintf(`
resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_eks_cluster.test.id
  disk_cpu_ratio    = 35
  subnets   	    = aws_subnet.test[*].id
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
`, rName))
}
