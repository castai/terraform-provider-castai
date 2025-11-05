package castai

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCloudAgnostic_ResourceEdgeLocationAWS(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "core-tf-acc-21-08-2025"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckEdgeLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeLocationAWSConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test edge location"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "aws.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.0.account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.0.vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.0.security_group_id"),
					resource.TestCheckResourceAttr(resourceName, "aws.0.subnet_ids.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "oci.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources[resourceName].Primary.ID
					return fmt.Sprintf("%v/%v/%v", organizationID, clusterID, edgeLocationID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"aws.0.access_key_id",
					"aws.0.secret_access_key",
				},
			},
			{
				Config: testAccEdgeLocationAWSUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated edge location description"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.2.id", "us-east-1c"),
					resource.TestCheckResourceAttr(resourceName, "zones.2.name", "us-east-1c"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeLocationGCP(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "core-tf-acc-gcp-21-08-2025"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckEdgeLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeLocationGCPConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test GCP edge location"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-central1"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-central1-b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-central1-b"),
					resource.TestCheckResourceAttr(resourceName, "gcp.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp.0.project_id"),
					resource.TestCheckResourceAttr(resourceName, "aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "oci.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccEdgeLocationGCPUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated GCP edge location"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-central1-b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-central1-b"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeLocationOCI(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckEdgeLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeLocationOCIConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test OCI edge location"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-phoenix-1"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "1"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "PHX-AD-1"),
					resource.TestCheckResourceAttr(resourceName, "oci.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.0.tenancy_id"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.0.compartment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.0.vcn_id"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "gcp.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccEdgeLocationOCIUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated OCI edge location"),
				),
			},
		},
	})
}

func testAccEdgeLocationAWSConfig(rName, clusterName string) string {
	return testAccEdgeLocationAWSConfigWithParams(rName, clusterName, "Test edge location", []string{"us-east-1a", "us-east-1b"})
}

func testAccEdgeLocationAWSUpdated(rName, clusterName string) string {
	return testAccEdgeLocationAWSConfigWithParams(rName, clusterName, "Updated edge location description", []string{"us-east-1a", "us-east-1b", "us-east-1c"})
}

func testAccEdgeLocationAWSConfigWithParams(rName, clusterName, description string, zones []string) string {
	organizationID := testAccGetOrganizationID()

	zonesConfig := ""
	subnetConfig := ""
	for i, zone := range zones {
		zonesConfig += fmt.Sprintf(`
  zones {
    id   = %q
    name = %q
  }`, zone, zone)
		subnetConfig += fmt.Sprintf(`
      %q = "subnet-%08d"`, zone, 12345678+i)
	}

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[5]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-east-1"
%[3]s

  aws {
    account_id         = "123456789012"
    access_key_id      = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    vpc_id             = "vpc-12345678"
    security_group_id  = "sg-12345678"
    subnet_ids = {%[4]s
    }
    name_tag           = "test-edge-location"
  }
}
`, rName, description, zonesConfig, subnetConfig, organizationID))
}

func testAccEdgeLocationGCPConfig(rName, clusterName string) string {
	return testAccEdgeLocationGCPConfigWithParams(rName, clusterName, "Test GCP edge location", []string{"us-central1-a", "us-central1-b"}, []string{"edge-location", "castai"})
}

func testAccEdgeLocationGCPUpdated(rName, clusterName string) string {
	return testAccEdgeLocationGCPConfigWithParams(rName, clusterName, "Updated GCP edge location", []string{"us-central1-a", "us-central1-b"}, []string{"edge-location"})
}

func testAccEdgeLocationGCPConfigWithParams(rName, clusterName, description string, zones []string, networkTags []string) string {
	organizationID := testAccGetOrganizationID()
	projectID := "test-project-123456"

	zonesConfig := ""
	for _, zone := range zones {
		zonesConfig += fmt.Sprintf(`
  zones {
    id   = %q
    name = %q
  }`, zone, zone)
	}

	networkTagsConfig := ""
	for i, tag := range networkTags {
		if i > 0 {
			networkTagsConfig += ", "
		}
		networkTagsConfig += fmt.Sprintf("%q", tag)
	}

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[6]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-central1"
%[3]s

  gcp {
    project_id                     = %[7]q
    client_service_account_json    = base64encode(jsonencode({
      "type": "service_account",
      "project_id": %[7]q,
      "private_key_id": "key123",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END PRIVATE KEY-----\n",
      "client_email": "test@test-project-123456.iam.gserviceaccount.com",
      "client_id": "123456789",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token"
    }))
    network_name                   = "test-network"
    subnet_name                    = "test-subnet"
    network_tags                   = [%[4]s]
  }
}
`, rName, description, zonesConfig, networkTagsConfig, organizationID, organizationID, projectID))
}

func testAccEdgeLocationOCIConfig(rName string) string {
	return testAccEdgeLocationOCIConfigWithParams(rName, "Test OCI edge location")
}

func testAccEdgeLocationOCIUpdated(rName string) string {
	return testAccEdgeLocationOCIConfigWithParams(rName, "Updated OCI edge location")
}

func testAccEdgeLocationOCIConfigWithParams(rName, description string) string {
	organizationID := testAccGetOrganizationID()
	clusterName := "test-oci-cluster"

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[3]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-phoenix-1"

  zones {
    id   = "PHX-AD-1"
    name = "PHX-AD-1"
  }

  oci {
    tenancy_id     = "ocid1.tenancy.oc1..example"
    compartment_id = "ocid1.compartment.oc1..example"
    user_id        = "ocid1.user.oc1..example"
    fingerprint    = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
    private_key    = base64encode("-----BEGIN RSA PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END RSA PRIVATE KEY-----\n")
    vcn_id         = "ocid1.vcn.oc1.phx.example"
    subnet_id      = "ocid1.subnet.oc1.phx.example"
  }
}
`, rName, description, organizationID))
}

func testAccCheckEdgeLocationDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_edge_location" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.Attributes["cluster_id"]
		id := rs.Primary.ID

		response, err := client.EdgeLocationsAPIGetEdgeLocationWithResponse(ctx, organizationID, clusterID, id)
		if err != nil {
			return err
		}
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("edge location %s still exists", rs.Primary.ID)
	}

	return nil
}

func testOmniClusterConfig(clusterName string) string {
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
