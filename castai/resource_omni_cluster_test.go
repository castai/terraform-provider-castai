package castai

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"net/http"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

func TestAccCloudAgnostic_ResourceOmniCluster(t *testing.T) {
	resourceName := "castai_omni_cluster.test"
	clusterName := fmt.Sprintf("omni-tf-acc-cluster-%v", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOmniClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOmniClusterConfig(clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "organization_id", testAccGetOrganizationID()),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
				),
			},
		},
	})
}

func testAccCheckOmniClusterDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniAPI
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_omni_cluster" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.ID

		response, err := client.ClustersAPIGetClusterWithResponse(ctx, organizationID, clusterID, nil)
		if err != nil {
			return err
		}
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("omni cluster %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccCleanupEdgeLocationsAndCluster(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniAPI
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_omni_cluster" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.ID

		if err := cleanupRemainingEdgeLocations(ctx, client, organizationID, clusterID); err != nil {
			return err
		}

		b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
		err := backoff.Retry(func() error {
			resp, err := client.ClustersAPIDeleteClusterWithResponse(ctx, organizationID, clusterID)
			if err != nil {
				return err
			}
			if resp.StatusCode() == http.StatusNotFound || resp.StatusCode() == http.StatusOK {
				return nil
			}
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode(), string(resp.Body))
		}, b)
		if err != nil {
			return fmt.Errorf("failed to delete omni cluster %s: %w", clusterID, err)
		}
	}

	return nil
}

func cleanupRemainingEdgeLocations(ctx context.Context, client omni.ClientWithResponsesInterface, organizationID, clusterID string) error {
	listResp, err := client.EdgeLocationsAPIListEdgeLocationsWithResponse(ctx, organizationID, clusterID, nil)
	if err != nil {
		return fmt.Errorf("listing edge locations: %w", err)
	}
	if listResp.StatusCode() != http.StatusOK || listResp.JSON200 == nil || listResp.JSON200.Items == nil {
		return nil
	}

	for _, el := range *listResp.JSON200.Items {
		if el.Id == nil {
			continue
		}
		elID := *el.Id
		if _, err := client.EdgeLocationsAPIDeleteEdgeLocationWithResponse(ctx, organizationID, clusterID, elID); err != nil {
			return fmt.Errorf("deleting edge location %s: %w", elID, err)
		}

		ticker := time.NewTicker(5 * time.Second)
		deleted := false
		for !deleted {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return ctx.Err()
			case <-ticker.C:
				getResp, err := client.EdgeLocationsAPIGetEdgeLocationWithResponse(ctx, organizationID, clusterID, elID)
				if err != nil {
					ticker.Stop()
					return fmt.Errorf("polling edge location %s: %w", elID, err)
				}
				if getResp.StatusCode() == http.StatusNotFound {
					deleted = true
				}
			}
		}
		ticker.Stop()
	}

	return nil
}

func testAccCheckEdgeResourcesDestroy(checks ...resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, check := range checks {
			if check != nil {
				if err := check(s); err != nil {
					return err
				}
			}
		}
		return testAccCleanupEdgeLocationsAndCluster(s)
	}
}

func testAccOmniClusterConfig(clusterName string) string {
	organizationID := testAccGetOrganizationID()
	return fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id = "test-project-123456"
  location   = "us-central1-c"
  name       = %[2]q
}

resource "castai_omni_cluster" "test" {
  organization_id = %[1]q
  cluster_id      = castai_gke_cluster.test.id
}
`, organizationID, clusterName)
}
