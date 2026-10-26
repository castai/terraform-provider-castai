package castai

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccGKE_ClusterImpersonation onboards a pre-existing GKE cluster to CAST AI
// using service account impersonation instead of a service account key.
//
// Required environment variables:
//   - CASTAI_API_TOKEN, ACCEPTANCE_TEST_ORGANIZATION_ID (testAccPreCheck)
//   - GOOGLE_PROJECT_ID - GCP project containing the shared acceptance cluster
//   - GOOGLE_CLUSTER_NAME - name of the pre-existing GKE cluster
//   - GOOGLE_CLUSTER_LOCATION - zone or region of the cluster, e.g. us-central1-c
func TestAccGKE_ClusterImpersonation(t *testing.T) {
	projectID := os.Getenv("GOOGLE_PROJECT_ID")
	clusterName := os.Getenv("GOOGLE_CLUSTER_NAME")
	location := os.Getenv("GOOGLE_CLUSTER_LOCATION")
	if projectID == "" || clusterName == "" || location == "" {
		t.Skip("GOOGLE_PROJECT_ID, GOOGLE_CLUSTER_NAME and GOOGLE_CLUSTER_LOCATION must be set for this acceptance test")
	}

	rName := fmt.Sprintf("%v-gke-imp-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_gke_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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
		Steps: []resource.TestStep{
			{
				Config: testAccGKEImpersonationConfig(rName, clusterName, location, projectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_token"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials_id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttr(resourceName, "name", clusterName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "location", location),
					resource.TestCheckResourceAttr(resourceName, "credentials.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.0.target_service_account_email"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.0.castai_service_account_email"),
					resource.TestCheckResourceAttrPair(
						resourceName, "credentials.0.castai_service_account_email",
						"data.castai_impersonation_service_account.gke", "email",
					),
				),
			},
			{
				Config:   testAccGKEImpersonationConfig(rName, clusterName, location, projectID),
				PlanOnly: true,
			},
		},
	})
}

func testAccGKEImpersonationConfig(rName, clusterName, location, projectID string) string {
	return fmt.Sprintf(`
data "castai_impersonation_service_account" "gke" {
  provider_type = "gke"
}

data "castai_gke_user_policies" "gke" {}

resource "google_service_account" "target" {
  account_id   = "%[1]s"
  display_name = "Terraform acceptance tests GKE impersonation target for %[2]s"
  project      = %[4]q
}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = "castai.tfAccGkeImp.${substr(sha1("%[1]s"), 0, 8)}"
  title       = "Terraform acceptance tests GKE impersonation role for %[2]s"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gke_user_policies.gke.policy)
  project     = %[4]q
  stage       = "GA"
}

resource "google_project_iam_member" "castai_role_binding" {
  project = %[4]q
  role    = google_project_iam_custom_role.castai_role.id
  member  = "serviceAccount:${google_service_account.target.email}"
}

resource "google_service_account_iam_member" "token_creator_binding" {
  service_account_id = google_service_account.target.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${data.castai_impersonation_service_account.gke.email}"
}

resource "castai_gke_cluster" "test" {
  project_id                 = %[4]q
  location                   = %[3]q
  name                       = %[2]q
  delete_nodes_on_disconnect = false

  credentials {
    target_service_account_email = google_service_account.target.email
    castai_service_account_email = data.castai_impersonation_service_account.gke.email
  }

  depends_on = [
    google_project_iam_member.castai_role_binding,
    google_service_account_iam_member.token_creator_binding,
  ]
}
`, rName, clusterName, location, projectID)
}
