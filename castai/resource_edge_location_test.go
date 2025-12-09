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

func TestAccCloudAgnostic_ResourceEdgeLocationAWS(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "omni-tf-acc-aws"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeLocationDestroy,
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
					resource.TestCheckResourceAttrSet(resourceName, "aws.account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.security_group_id"),
					resource.TestCheckResourceAttr(resourceName, "aws.subnet_ids.%", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "1"),
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
					"aws.access_key_id_wo",
					"aws.secret_access_key_wo",
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
			{
				Config: testAccEdgeLocationAWSCredentialsUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "2"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeLocationGCP(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "omni-tf-acc-gcp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeLocationDestroy,
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
					resource.TestCheckResourceAttrSet(resourceName, "gcp.project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "1"),
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
			{
				Config: testAccEdgeLocationGCPCredentialsUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "2"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeLocationOCI(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeLocationDestroy,
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
					resource.TestCheckResourceAttrSet(resourceName, "oci.tenancy_id"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.compartment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "oci.subnet_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "1"),
				),
			},
			{
				Config: testAccEdgeLocationOCIUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated OCI edge location"),
				),
			},
			{
				Config: testAccEdgeLocationOCICredentialsUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "2"),
				),
			},
		},
	})
}

func testAccEdgeLocationAWSConfig(rName, clusterName string) string {
	return testAccEdgeLocationAWSConfigWithParams(rName, clusterName, "Test edge location", []string{"us-east-1a", "us-east-1b"},
		`access_key_id_wo     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`)
}

func testAccEdgeLocationAWSUpdated(rName, clusterName string) string {
	return testAccEdgeLocationAWSConfigWithParams(rName, clusterName, "Updated edge location description", []string{"us-east-1a", "us-east-1b", "us-east-1c"},
		`access_key_id_wo     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`)
}

func testAccEdgeLocationAWSCredentialsUpdated(rName, clusterName string) string {
	return testAccEdgeLocationAWSConfigWithParams(rName, clusterName, "Updated edge location description", []string{"us-east-1a", "us-east-1b", "us-east-1c"},
		`access_key_id_wo     = "AKIAIOSFODNN7NEWKEY1"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYNEWKEY12345"`)
}

func testAccEdgeLocationAWSConfigWithParams(rName, clusterName, description string, zones []string, awsCredentials string) string {
	organizationID := testAccGetOrganizationID()

	zonesConfig := "zones = ["
	subnetConfig := ""
	for i, zone := range zones {
		if i > 0 {
			zonesConfig += ", "
		}
		zonesConfig += fmt.Sprintf(`{
    id   = %q
    name = %q
  }`, zone, zone)
		subnetConfig += fmt.Sprintf(`
      %q = "subnet-%08d"`, zone, 12345678+i)
	}
	zonesConfig += "]"

	return concatenateConfigs(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[5]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-east-1"
%[3]s

  aws = {
    account_id           = "123456789012"
    %[6]s
    vpc_id               = "vpc-12345678"
    security_group_id    = "sg-12345678"
    subnet_ids = {%[4]s
    }
    name_tag             = "test-edge-location"
  }
}
`, rName, description, zonesConfig, subnetConfig, organizationID, awsCredentials))
}

func testAccEdgeLocationGCPConfig(rName, clusterName string) string {
	return testAccEdgeLocationGCPConfigWithParams(rName, clusterName,
		"Test GCP edge location", []string{"us-central1-a", "us-central1-b"}, []string{"edge-location", "castai"},
		`client_service_account_json_base64_wo = base64encode(jsonencode({
      "type": "service_account",
      "project_id": "test-project-123456",
      "private_key_id": "key123",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END PRIVATE KEY-----\n",
      "client_email": "test@test-project-123456.iam.gserviceaccount.com",
      "client_id": "123456789",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token"
    }))`)
}

func testAccEdgeLocationGCPUpdated(rName, clusterName string) string {
	return testAccEdgeLocationGCPConfigWithParams(rName, clusterName,
		"Updated GCP edge location", []string{"us-central1-a", "us-central1-b"}, []string{"edge-location", "castai"},
		`client_service_account_json_base64_wo = base64encode(jsonencode({
      "type": "service_account",
      "project_id": "test-project-123456",
      "private_key_id": "key123",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END PRIVATE KEY-----\n",
      "client_email": "test@test-project-123456.iam.gserviceaccount.com",
      "client_id": "123456789",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token"
    }))`)
}

func testAccEdgeLocationGCPCredentialsUpdated(rName, clusterName string) string {
	return testAccEdgeLocationGCPConfigWithParams(rName, clusterName,
		"Updated GCP edge location", []string{"us-central1-a", "us-central1-b"}, []string{"edge-location", "castai"},
		`client_service_account_json_base64_wo = base64encode(jsonencode({
      "type": "service_account",
      "project_id": "test-project-123456",
      "private_key_id": "key456-new",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...NEWKEY...==\n-----END PRIVATE KEY-----\n",
      "client_email": "test@test-project-123456.iam.gserviceaccount.com",
      "client_id": "123456789",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token"
    }))`)
}

func testAccEdgeLocationGCPConfigWithParams(rName, clusterName, description string, zones []string, networkTags []string, gcpCredentials string) string {
	organizationID := testAccGetOrganizationID()
	zonesConfig := "zones = ["
	for i, zone := range zones {
		if i > 0 {
			zonesConfig += ", "
		}
		zonesConfig += fmt.Sprintf(`{
    id   = %q
    name = %q
  }`, zone, zone)
	}
	zonesConfig += "]"

	networkTagsConfig := ""
	for i, tag := range networkTags {
		if i > 0 {
			networkTagsConfig += ", "
		}
		networkTagsConfig += fmt.Sprintf("%q", tag)
	}

	return concatenateConfigs(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[6]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-central1"
%[3]s

  gcp = {
    project_id     = "test-project-123456"
    %[5]s
    network_name   = "test-network"
    subnet_name    = "test-subnet"
    network_tags   = [%[4]s]
  }
}
`, rName, description, zonesConfig, networkTagsConfig, gcpCredentials, organizationID))
}

func testAccEdgeLocationOCIConfig(rName string) string {
	return testAccEdgeLocationOCIConfigWithParams(rName, "Test OCI edge location",
		`user_id_wo      = "ocid1.user.oc1..example"
    fingerprint_wo  = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
    private_key_base64_wo  = base64encode("-----BEGIN RSA PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END RSA PRIVATE KEY-----\n")`)
}

func testAccEdgeLocationOCIUpdated(rName string) string {
	return testAccEdgeLocationOCIConfigWithParams(rName, "Updated OCI edge location",
		`user_id_wo      = "ocid1.user.oc1..example"
    fingerprint_wo  = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
    private_key_base64_wo  = base64encode("-----BEGIN RSA PRIVATE KEY-----\nMIIE...EXAMPLE...==\n-----END RSA PRIVATE KEY-----\n")`)
}

func testAccEdgeLocationOCICredentialsUpdated(rName string) string {
	return testAccEdgeLocationOCIConfigWithParams(rName, "Updated OCI edge location",
		`user_id_wo      = "ocid1.user.oc1..example"
    fingerprint_wo  = "11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00"
    private_key_base64_wo  = base64encode("-----BEGIN RSA PRIVATE KEY-----\nMIIE...NEWKEY...==\n-----END RSA PRIVATE KEY-----\n")`)
}

func testAccEdgeLocationOCIConfigWithParams(rName, description, ociCredentials string) string {
	organizationID := testAccGetOrganizationID()
	clusterName := "test-oci-cluster"

	return concatenateConfigs(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id = %[3]q
  cluster_id      = castai_omni_cluster.test.id
  name            = %[1]q
  description     = %[2]q
  region          = "us-phoenix-1"
  zones = [{
    id   = "1"
    name = "PHX-AD-1"
  }]

  oci = {
    tenancy_id      = "ocid1.tenancy.oc1..example"
    compartment_id  = "ocid1.compartment.oc1..example"
    %[4]s
    vcn_id          = "ocid1.vcn.oc1.phx.example"
    subnet_id       = "ocid1.subnet.oc1.phx.example"
  }
}
`, rName, description, organizationID, ociCredentials))
}

func testAccCheckEdgeLocationDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniAPI
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
