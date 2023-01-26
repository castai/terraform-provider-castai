package castai

import (
	"bytes"
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
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
				"shouldTaint": false,
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
constraints.% = 9
constraints.compute_optimized = false
constraints.fallback_restore_rate_seconds = 0
constraints.gpu = {"exclude_names":[],"include_names":[],"manufacturers":["NVIDIA"]}
constraints.instance_families = {"exclude":["p4d","p3dn","p2","g3s","g5g","g5","g3"],"include":[]}
constraints.max_cpu = 10000
constraints.min_cpu = 10
constraints.spot = false
constraints.storage_optimized = false
constraints.use_spot_fallbacks = false
custom_label.% = 0
name = gpu
rebalancing_config_min_nodes = 0
should_taint = false
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckNodeTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNodeTemplateConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "should_taint", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "should_taint", "false"),
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
	return ConfigCompose(testAccClusterConfig(rName, clusterName), testAccNodeConfig(rName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_clusterid.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = true
		}
	`, rName))
}

func testNodeTemplateUpdated(rName, clusterName string) string {
	return ConfigCompose(testAccClusterConfig(rName, clusterName), testAccNodeConfig(rName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			cluster_id        = castai_eks_clusterid.test.id
			name = %[1]q
			configuration_id = castai_node_configuration.test.id
			should_taint = false
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
