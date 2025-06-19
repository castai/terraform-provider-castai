package castai

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TODO (romank): check how this test should be run

func TestAccResourceAllocationGroup(t *testing.T) {
	resourceName := "castai_resource_allocation_group.test"
	//projectID := os.Getenv("GOOGLE_PROJECT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAllocationGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: allocationGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "ag_example"),
					resource.TestCheckResourceAttr(resourceName, "cluster_ids.0", "cluster-123"),
					resource.TestCheckResourceAttr(resourceName, "cluster_ids.1", "cluster-456"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.0", "namespace-a"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.1", "namespace-b"),
					resource.TestCheckResourceAttr(resourceName, "labels.environment", "prod-master"),
					resource.TestCheckResourceAttr(resourceName, "labels.team", "cost-report"),
					resource.TestCheckResourceAttr(resourceName, `labels["app.kubernetes.io/name"]`, "app"),
					resource.TestCheckResourceAttr(resourceName, "labels_operator", "AND"),
				),
			},
		},
	})
}

func testAccCheckAllocationGroupDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_allocation_group" {
			continue
		}

		id := rs.Primary.ID
		resp, err := client.WorkloadOptimizationAPIGetAllocationGroupWithResponse(ctx, id)
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("scaling policy %s still exists", rs.Primary.ID)
	}

	return nil
}

func allocationGroupConfig() string {
	// TODO (romank): put real cluster id when testing
	cfg := fmt.Sprintf(`
	resource "castai_allocation_group" "ag_example" {
	  cluster_ids = [
		"1a58d6b4-bc0e-4417-b9c7-31d15c313f3f",
		"d204b988-5db5-472e-a258-bf763a0f4a93"
	  ]
	
	  namespaces = [
		"namespace-a",
		"namespace-b"
	  ]
	
	  labels = {
		environment = "production",
		team = "my-team",
		"app.kubernetes.io/name" = "app-name"
	  }
	
	  labels_operator = "AND"
	}
	`)

	return ConfigCompose(cfg)

}
