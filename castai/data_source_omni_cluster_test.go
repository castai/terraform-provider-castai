package castai

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudAgnostic_DataSourceOmniCluster(t *testing.T) {
	clusterName := "omni-tf-acc-gcp"
	dataSourceName := "data.castai_omni_cluster.test"
	resourceName := "castai_omni_cluster.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOmniClusterDataSourceConfig(clusterName),
				Check: resource.ComposeTestCheckFunc(
					// Verify data source ID matches the resource ID
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					// Verify organization_id is correctly passed through
					resource.TestCheckResourceAttr(dataSourceName, "organization_id", testAccGetOrganizationID()),
					// Verify cluster metadata fields are populated
					resource.TestCheckResourceAttrSet(dataSourceName, "name"),
					resource.TestCheckResourceAttrSet(dataSourceName, "state"),
					// Verify OIDC config fields have expected formats
					resource.TestMatchResourceAttr(dataSourceName, "castai_oidc_config.gcp_service_account_email",
						regexp.MustCompile(`^.+@.+\.iam\.gserviceaccount\.com$`)),
					resource.TestMatchResourceAttr(dataSourceName, "castai_oidc_config.gcp_service_account_unique_id",
						regexp.MustCompile(`^\d+$`)),
				),
			},
		},
	})
}

func testAccOmniClusterDataSourceConfig(clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
data "castai_omni_cluster" "test" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test.id
}
`, organizationID))
}
