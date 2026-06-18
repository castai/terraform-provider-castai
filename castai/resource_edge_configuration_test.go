package castai

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCloudAgnostic_ResourceEdgeConfigurationGCP(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "omni-tf-acc-gcp"
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationGCPConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "gcp.image_id", "projects/castai/global/images/castai-edge-v1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
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
				Config: testAccEdgeConfigurationGCPUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "gcp.image_id", "projects/castai/global/images/castai-edge-v2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "gcp.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeConfigurationAWS(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "omni-tf-acc-aws-cfg"
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationAWSConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "aws.image_id", "ami-0abcdef1234567890"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "aws.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
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
				Config: testAccEdgeConfigurationAWSUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "aws.image_id", "ami-0updated1234567890"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "aws.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeConfigurationOCI(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "test-oci-cluster-cfg"
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationOCIConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "oci.image_id", "ocid1.image.oc1.phx.example"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "oci.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
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
				Config: testAccEdgeConfigurationOCIUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "oci.image_id", "ocid1.image.oc1.phx.updated"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "oci.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func testAccEdgeConfigurationGCPConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  gcp = {
    image_id          = "projects/castai/global/images/castai-edge-v1"
    boot_disk_size_gib = 100
    labels = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationGCPUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  gcp = {
    image_id          = "projects/castai/global/images/castai-edge-v2"
    boot_disk_size_gib = 200
    labels = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationAWSConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  aws = {
    image_id          = "ami-0abcdef1234567890"
    boot_disk_size_gib = 100
    tags = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationAWSUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  aws = {
    image_id          = "ami-0updated1234567890"
    boot_disk_size_gib = 200
    tags = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationOCIConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationOCIConfig(rName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  oci = {
    image_id          = "ocid1.image.oc1.phx.example"
    boot_disk_size_gib = 100
    tags = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationOCIUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationOCIConfig(rName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  oci = {
    image_id          = "ocid1.image.oc1.phx.updated"
    boot_disk_size_gib = 200
    tags = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccCheckEdgeConfigurationDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniAPI
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_edge_configuration" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.Attributes["cluster_id"]
		edgeLocationID := rs.Primary.Attributes["edge_location_id"]
		configID := rs.Primary.ID

		response, err := client.EdgeConfigurationsAPIGetEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configID, nil)
		if err != nil {
			return err
		}
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("edge configuration %s still exists", rs.Primary.ID)
	}

	return nil
}
