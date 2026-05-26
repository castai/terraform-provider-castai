package castai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCloudAgnostic_ResourceEdgeConfigurationDefault(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "omni-tf-acc-gcp"
	resourceName := "castai_edge_configuration_default.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationDefaultConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "castai_edge_configuration.test.id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "configuration_id"),
					resource.TestCheckResourceAttr(resourceName, "name", "test-edge-config"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "gcp"),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd.sock"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources["castai_edge_location.test"].Primary.ID
					configID := s.RootModule().Resources["castai_edge_configuration.test"].Primary.ID
					return fmt.Sprintf("%s/%s/%s/%s", organizationID, clusterID, edgeLocationID, configID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEdgeConfigurationDefaultUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "test-edge-config-updated"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "gcp"),
					resource.TestCheckResourceAttrSet(resourceName, "configuration_id"),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func testAccEdgeConfigurationDefaultConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name           = "test-edge-config"

  gcp = {
    image_id = "projects/castai/global/images/castai-edge-v1"
  }

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }
}

resource "castai_edge_configuration_default" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  configuration_id = castai_edge_configuration.test.id
}
`, organizationID),
	)
}

func testAccEdgeConfigurationDefaultUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name           = "test-edge-config"

  gcp = {
    image_id = "projects/castai/global/images/castai-edge-v1"
  }
}

resource "castai_edge_configuration" "test2" {
  organization_id = %[1]q
  cluster_id      = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name           = "test-edge-config-updated"

  gcp = {
    image_id = "projects/castai/global/images/castai-edge-v2"
  }

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }
}

resource "castai_edge_configuration_default" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  configuration_id = castai_edge_configuration.test2.id
}
`, organizationID),
	)
}
