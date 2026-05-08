package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id 	 = %[3]q
  cluster_id      	 = castai_omni_cluster.test.id
  name            	 = %[1]q
  description     	 = %[2]q
  region          	 = "us-phoenix-1"
  control_plane_mode = "SHARED"
  zones = [{
    id   = "1"
    name = "PHX-AD-1"
  }]

  oci = {
    tenancy_id        = "ocid1.tenancy.oc1..example"
    compartment_id    = "ocid1.compartment.oc1..example"
    %[4]s
    vcn_id            = "ocid1.vcn.oc1.phx.example"
    vcn_cidr          = "10.0.0.0/16"
    subnet_id         = "ocid1.subnet.oc1.phx.example"
    security_group_id = "ocid1.networksecuritygroup.oc1.phx.example"
  }
}
`, rName, description, organizationID, ociCredentials))
}

func TestAccCloudAgnostic_ResourceEdgeLocationAWSImpersonation(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "omni-tf-acc-aws"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test edge location impersonation"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-east-1b"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.account_id"),
					resource.TestCheckResourceAttr(resourceName, "aws.role_arn", "arn:aws:iam::123456789012:role/castai-omni-edge"),
					resource.TestCheckResourceAttr(resourceName, "aws.vpc_cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.vpc_peered"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aws.security_group_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "networking"),
					resource.TestCheckNoResourceAttr(resourceName, "control_plane"),
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
			},
			{
				Config: testAccEdgeLocationAWSImpersonationUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated edge location impersonation"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-east-1a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-east-1b"),
					resource.TestCheckResourceAttr(resourceName, "zones.2.id", "us-east-1c"),
					resource.TestCheckResourceAttr(resourceName, "zones.2.name", "us-east-1c"),
					resource.TestCheckResourceAttr(resourceName, "aws.role_arn", "arn:aws:iam::123456789012:role/castai-omni-edge-updated"),
					resource.TestCheckResourceAttr(resourceName, "networking.tunneled_cidrs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "networking.tunneled_cidrs.0", "10.10.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "networking.tunneled_cidrs.1", "192.168.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "control_plane.ha", "false"),
				),
			},
			{
				Config: testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "networking"),
					resource.TestCheckNoResourceAttr(resourceName, "control_plane"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeLocationGCPImpersonation(t *testing.T) {
	rName := fmt.Sprintf("%v-edge-loc-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_edge_location.test"
	clusterName := "omni-tf-acc-gcp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test GCP edge location impersonation"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-central1"),
					resource.TestCheckResourceAttr(resourceName, "zones.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.id", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.0.name", "us-central1-a"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.id", "us-central1-b"),
					resource.TestCheckResourceAttr(resourceName, "zones.1.name", "us-central1-b"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp.project_id"),
					resource.TestCheckResourceAttr(resourceName, "gcp.target_service_account_email", "castai-omni@test-project-123456.iam.gserviceaccount.com"),
					resource.TestCheckResourceAttr(resourceName, "gcp.subnet_cidr", "10.0.0.0/20"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "credentials_revision", "1"),
				),
			},
			{
				Config: testAccEdgeLocationGCPImpersonationUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated GCP edge location impersonation"),
					resource.TestCheckResourceAttr(resourceName, "gcp.target_service_account_email", "castai-omni-updated@test-project-123456.iam.gserviceaccount.com"),
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

func formatAWSZonesAndSubnets(zones []string) (zonesConfig string, subnetConfig string) {
	var zonesBuilder strings.Builder
	var subnetBuilder strings.Builder

	zonesBuilder.WriteString("zones = [")
	for i, zone := range zones {
		if i > 0 {
			zonesBuilder.WriteString(", ")
		}
		fmt.Fprintf(&zonesBuilder, `{
    id   = %q
    name = %q
	  }`, zone, zone)
		fmt.Fprintf(&subnetBuilder, `
	      %q = "subnet-%08d"`, zone, 12345678+i)
	}
	zonesBuilder.WriteString("]")

	zonesConfig = zonesBuilder.String()
	subnetConfig = subnetBuilder.String()

	return
}

func testAccEdgeLocationAWSImpersonationConfig(rName, clusterName string) string {
	return testAccEdgeLocationAWSImpersonationConfigWithParams(rName, clusterName, "Test edge location impersonation",
		[]string{"us-east-1a", "us-east-1b"}, "arn:aws:iam::123456789012:role/castai-omni-edge", nil, nil)
}

func testAccEdgeLocationAWSImpersonationUpdated(rName, clusterName string) string {
	ha := false
	return testAccEdgeLocationAWSImpersonationConfigWithParams(rName, clusterName, "Updated edge location impersonation",
		[]string{"us-east-1a", "us-east-1b", "us-east-1c"}, "arn:aws:iam::123456789012:role/castai-omni-edge-updated",
		[]string{"10.10.0.0/16", "192.168.0.0/24"}, &ha)
}

func testAccEdgeLocationAWSImpersonationConfigWithParams(rName, clusterName, description string, zones []string, roleArn string, tunneledCIDRs []string, ha *bool) string {
	organizationID := testAccGetOrganizationID()

	zonesConfig, subnetConfig := formatAWSZonesAndSubnets(zones)

	networkingBlock := ""
	if tunneledCIDRs != nil {
		quoted := make([]string, 0, len(tunneledCIDRs))
		for _, c := range tunneledCIDRs {
			quoted = append(quoted, fmt.Sprintf("%q", c))
		}
		networkingBlock = fmt.Sprintf(`
  networking = {
    tunneled_cidrs = [%s]
  }`, strings.Join(quoted, ", "))
	}

	controlPlaneBlock := ""
	if ha != nil {
		controlPlaneBlock = fmt.Sprintf(`
  control_plane = {
    ha = %t
  }`, *ha)
	}

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id 	 = %[5]q
  cluster_id      	 = castai_omni_cluster.test.id
  name            	 = %[1]q
  description     	 = %[2]q
  region          	 = "us-east-1"
  control_plane_mode = "SHARED"
%[3]s
%[7]s
%[8]s

  aws = {
    account_id               = "123456789012"
    role_arn                 = "%[6]s"
    vpc_id                   = "vpc-12345678"
    vpc_peered               = true
    instance_service_account = "arn:aws:iam::123456789012:role/castai-omni-edge"
    vpc_cidr                 = "10.0.0.0/16"
    security_group_id        = "sg-12345678"
    subnet_ids = {%[4]s
    }
  }
}
`, rName, description, zonesConfig, subnetConfig, organizationID, roleArn, networkingBlock, controlPlaneBlock))
}

func testAccEdgeLocationGCPImpersonationConfig(rName, clusterName string) string {
	return testAccEdgeLocationGCPImpersonationConfigWithParams(rName, clusterName,
		"Test GCP edge location impersonation",
		"castai-omni@test-project-123456.iam.gserviceaccount.com",
		[]string{"us-central1-a", "us-central1-b"},
		[]string{"edge-location", "castai"})
}

func testAccEdgeLocationGCPImpersonationUpdated(rName, clusterName string) string {
	return testAccEdgeLocationGCPImpersonationConfigWithParams(rName, clusterName,
		"Updated GCP edge location impersonation",
		"castai-omni-updated@test-project-123456.iam.gserviceaccount.com",
		[]string{"us-central1-a", "us-central1-b"},
		[]string{"edge-location", "castai"})
}

func formatGCPZones(zones []string) string {
	var builder strings.Builder

	builder.WriteString("zones = [")
	for i, zone := range zones {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, `{
    id   = %q
    name = %q
	  }`, zone, zone)
	}
	builder.WriteString("]")
	return builder.String()
}

func formatGCPNetworkTags(networkTags []string) string {
	var builder strings.Builder

	for i, tag := range networkTags {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "%q", tag)
	}
	return builder.String()
}

func testAccEdgeLocationGCPImpersonationConfigWithParams(rName, clusterName, description, targetSA string, zones []string, networkTags []string) string {
	organizationID := testAccGetOrganizationID()

	zonesConfig := formatGCPZones(zones)
	networkTagsConfig := formatGCPNetworkTags(networkTags)

	return ConfigCompose(testOmniClusterConfig(clusterName), fmt.Sprintf(`
resource "castai_edge_location" "test" {
  organization_id 	 = %[5]q
  cluster_id      	 = castai_omni_cluster.test.id
  name            	 = %[1]q
  description     	 = %[2]q
  region          	 = "us-central1"
  control_plane_mode = "SHARED"
%[3]s

  gcp = {
    project_id                   = "test-project-123456"
	instance_service_account     = "custom-sa@test-project-123456.iam.gserviceaccount.com"
    target_service_account_email = "%[6]s"
    network_name                 = "test-network"
    subnet_name                  = "test-subnet"
    subnet_cidr                  = "10.0.0.0/20"
    network_tags                 = [%[4]s]
  }
}
`, rName, description, zonesConfig, networkTagsConfig, organizationID, targetSA))
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
  status = {
    omni_agent_version = "0.0.0"
    pod_cidr           = "10.244.0.0/16"
  }
}
`, organizationID, clusterName)
}
