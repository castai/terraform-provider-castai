package castai

import (
	"bytes"
	"context"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
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
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c369b1
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b1
configuration_id = 7dc4f922-29c9-4377-889c-0c8c5fb8d497
name = gpu
should_taint = false
Tainted = false
`, data.State().String())

}

//func TestAccResourceNodeTemplate_basic(t *testing.T) {
//	rName := fmt.Sprintf("%v-node-template-%v", ResourcePrefix, acctest.RandString(8))
//	resourceName := "castai_node_template.test"
//	clusterName := " zilvinas-01-24"
//
//	resource.ParallelTest(t, resource.TestCase{
//		PreCheck:          func() { testAccPreCheck(t) },
//		ProviderFactories: providerFactories,
//		CheckDestroy:      testAccCheckNodeTemplateDestroy,
//		Steps: []resource.TestStep{
//			{
//				Config: testAccNodeTemplateConfig(rName, clusterName),
//				Check:  resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(resourceName, "name", rName)),
//			},
//		},
//	})
//}
//
//func testAccNodeTemplateConfig(rName, clusterName string) string {
//	return ConfigCompose(testAccClusterConfig(rName, clusterName), fmt.Sprintf(`
//		resource "castai_node_template" "test" {
//			name = %[1]q
//			configuration_id = "f2840db8-522a-4abe-9653-28aaa4b086b4"
//			should_taint = true
//		}
//	`, rName))
//}
//
//func testAccCheckNodeTemplateDestroy(s *terraform.State) error {
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	client := testAccProvider.Meta().(*ProviderConfig).api
//	for _, rs := range s.RootModule().Resources {
//		if rs.Type != "castai_node_template" {
//			continue
//		}
//
//		id := rs.Primary.ID
//		clusterID := rs.Primary.Attributes["cluster_id"]
//		response, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID)
//		if err != nil {
//			return err
//		}
//		if response.StatusCode() == http.StatusNotFound {
//			return nil
//		}
//		if *response.JSON200.Items == nil {
//			// Should be no templates
//			return nil
//		}
//
//		return fmt.Errorf("node template %q still exists", id)
//	}
//
//	return nil
//}
