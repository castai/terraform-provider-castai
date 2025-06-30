package castai

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceAllocationGroup(t *testing.T) {
	resourceName := "castai_allocation_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAllocationGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidAllocationGroupConfig(),
				ExpectError: regexp.MustCompile(`allocation group must specify at least one of: cluster_ids, namespaces, or labels`),
			},
			{
				Config: allocationGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Test terraform example"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.0", "namespace-a"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.1", "namespace-b"),
					resource.TestCheckResourceAttr(resourceName, "labels.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "labels.team", "my-team"),
					resource.TestCheckResourceAttr(resourceName, "labels.app.kubernetes.io/name", "app-name"),
					resource.TestCheckResourceAttr(resourceName, "labels_operator", "AND"),
				),
			},
			{
				Config: allocationGroupUpdatedConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Test terraform example updated"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.0", "namespace-a"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.1", "namespace-b"),
					resource.TestCheckResourceAttr(resourceName, "namespaces.2", "namespace-c"),
					resource.TestCheckResourceAttr(resourceName, "labels.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "labels.team", "my-team"),
					resource.TestCheckResourceAttr(resourceName, "labels.app.kubernetes.io/name", "app-name-updated"),
					resource.TestCheckResourceAttr(resourceName, "labels_operator", "AND"),
				),
			},
			{
				// Import state by ID
				ImportState:       true,
				ResourceName:      "castai_allocation_group.test",
				ImportStateVerify: true,
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
		resp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, id)
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("allocation group %s still exists", rs.Primary.ID)
	}

	return nil
}

func invalidAllocationGroupConfig() string {
	cfg := `
	resource "castai_allocation_group" "test" {
		name = "Test terraform example"
	}`
	return ConfigCompose(cfg)
}

func allocationGroupConfig() string {
	cfg := `
	resource "castai_allocation_group" "test" {
      name = "Test terraform example"
	
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
	`

	return ConfigCompose(cfg)
}

func allocationGroupUpdatedConfig() string {
	cfg := `
	resource "castai_allocation_group" "test" {
      name = "Test terraform example updated"
	
	  namespaces = [
		"namespace-a",
		"namespace-b",
		"namespace-c"
	  ]
	
	  labels = {
		environment = "production",
		team = "my-team",
		"app.kubernetes.io/name" = "app-name-updated"
	  }
	
	  labels_operator = "AND"
	}
	`

	return ConfigCompose(cfg)
}
