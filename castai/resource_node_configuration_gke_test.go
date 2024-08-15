package castai

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	// acceptanceTestClusterSubnetworkName points to the subnet that the acceptance test cluster uses.
	// Actual value (should) be managed by our IaC repository and could be shared with other clusters as well.
	acceptanceTestClusterSubnetworkName = "ext-prov-e2e-shared-ip-range-nodes"
)

func TestAccResourceNodeConfiguration_gke(t *testing.T) {
	rName := fmt.Sprintf("%v-node-cfg-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_node_configuration.test"
	clusterName := "tf-core-acc-20230723"
	projectID := os.Getenv("GOOGLE_PROJECT_ID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckNodeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGKENodeConfigurationConfig(rName, clusterName, projectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "disk_cpu_ratio", "35"),
					resource.TestCheckResourceAttr(resourceName, "drain_timeout_sec", "10"),
					resource.TestCheckResourceAttr(resourceName, "min_disk_size", "122"),
					resource.TestCheckResourceAttr(resourceName, "aks.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.max_pods_per_node", "31"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.disk_type", "pd-balanced"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.network_tags.0", "ab"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.network_tags.1", "bc"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.zones.#", "0"),
				),
			},
			{
				Config: testAccGKENodeConfigurationUpdated(rName, clusterName, projectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "disk_cpu_ratio", "0"),
					resource.TestCheckResourceAttr(resourceName, "min_disk_size", "121"),
					resource.TestCheckResourceAttr(resourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.max_pods_per_node", "32"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.disk_type", "pd-ssd"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.network_tags.0", "bb"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.network_tags.1", "dd"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.zones.0", "us-central1-c"),
					resource.TestCheckResourceAttr(resourceName, "gke.0.use_ephemeral_storage_local_ssd", "true"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "> 4.75.0",
			},
			"google-beta": {
				Source:            "hashicorp/google-beta",
				VersionConstraint: "> 4.75.0",
			},
		},
	})
}

func testAccGKENodeConfigurationConfig(rName, clusterName, projectID string) string {
	return ConfigCompose(testAccGKEClusterConfig(rName, clusterName, projectID), fmt.Sprintf(`
resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_gke_cluster.test.id
  disk_cpu_ratio    = 35
  drain_timeout_sec = 10
  min_disk_size     = 122
  subnets   	    = [local.subnet_id]
  tags = {
    env = "development"
  }
  gke {
	max_pods_per_node = 31
    network_tags = ["ab", "bc"]
    disk_type = "pd-balanced"
  }
}

resource "castai_node_configuration_default" "test" {
  cluster_id       = castai_gke_cluster.test.id
  configuration_id = castai_node_configuration.test.id
}
`, rName))
}

func testAccGKENodeConfigurationUpdated(rName, clusterName, projectID string) string {
	return ConfigCompose(testAccGKEClusterConfig(rName, clusterName, projectID), fmt.Sprintf(`
resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_gke_cluster.test.id
  disk_cpu_ratio    = 0
  min_disk_size     = 121
  subnets   	    = [local.subnet_id]
  tags = {
    env = "development"
  }
  gke {
	max_pods_per_node = 32
    network_tags = ["bb", "dd"]
    disk_type = "pd-ssd"
    zones = ["us-central1-c"]
    use_ephemeral_storage_local_ssd = true
  }
}
`, rName))
}
func testAccGKEClusterConfig(rName string, clusterName string, projectID string) string {
	return ConfigCompose(testAccGCPConfig(rName, clusterName, projectID), fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id                 = %[1]q
  location                   = "us-central1-c" 
  name                       = %[2]q 
  credentials_json           = base64decode(google_service_account_key.castai_key.private_key)
}

`, projectID, clusterName))
}

func testAccGCPConfig(rName, clusterName, projectID string) string {

	return fmt.Sprintf(`

locals {
  service_account_id    = %[3]q
  cluster_name = %[1]q
  service_account_email = "${local.service_account_id}@$%[2]s.iam.gserviceaccount.com"
  custom_role_id        = "castai.tfAcc.${substr(sha1(local.service_account_id),0,8)}.tf"
  subnet_id = "projects/%[2]s/regions/us-central1/subnetworks/%[4]s"
}

resource "google_service_account" "castai_service_account" {
  account_id   = local.service_account_id
  display_name = "Terraform acceptance tests for GKE cluster %[1]s"
  project      = %[2]q
}

data "castai_gke_user_policies" "gke" {}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = local.custom_role_id
  title       = "Terraform acceptance tests for GKE cluster %[1]s"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gke_user_policies.gke.policy)
  project     = %[2]q
  stage       = "GA"
}

resource "google_project_iam_member" "project_developer" {
  project = %[2]q
  role    = "roles/container.developer"
  member = google_service_account.castai_service_account.member
}

resource "google_project_iam_member" "project_sa" {
  project = %[2]q
  role    = "roles/iam.serviceAccountUser"
  member = google_service_account.castai_service_account.member
}

resource "google_project_iam_member" "project_castai" {
  project = %[2]q
  role    = "projects/%[2]s/roles/${local.custom_role_id}"
  member = google_service_account.castai_service_account.member
}

resource "google_service_account_key" "castai_key" {
  service_account_id = google_service_account.castai_service_account.name
  public_key_type    = "TYPE_X509_PEM_FILE"
}

`, clusterName, projectID, rName, acceptanceTestClusterSubnetworkName)
}
