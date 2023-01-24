package castai

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"net/http"
	"testing"
	"time"
)

func TestAccResourceNodeTemplate_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-node-template-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_node_template.test"
	clusterName := " zilvinas-01-24"

	resource.ParallelTest(t, resource.TestCase{
		//PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckNodeTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNodeTemplateConfig(rName, clusterName),
				Check:  resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(resourceName, "name", rName)),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 4.0",
			},
		}})
}

func testAccNodeTemplateConfig(rName, clusterName string) string {
	return ConfigCompose(testAccClusterConfig(rName, clusterName), fmt.Sprintf(`
		resource "castai_node_template" "test" {
			name = %[1]q
			configuration_id = "f2840db8-522a-4abe-9653-28aaa4b086b4"
			should_taint = true
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
		if *response.JSON200.Items == nil {
			// Should be no templates
			return nil
		}

		return fmt.Errorf("node template %q still exists", id)
	}

	return nil
}
