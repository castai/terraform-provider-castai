package castai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCloudAgnostic_Commitment(t *testing.T) {
	rName := fmt.Sprintf("%v-commitment-%v", ResourcePrefix, acctest.RandString(8))
	cudID := acctest.RandStringFromCharSet(12, "0123456789")
	resourceName := "castai_commitment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with auto_assignment not set — server decides.
				Config: testAccCommitmentConfig(rName, cudID, "INACTIVE", 1.0, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "cloud", "GCP"),
					resource.TestCheckResourceAttr(resourceName, "region", "us-central1"),
					resource.TestCheckResourceAttr(resourceName, "type", "RESOURCE_CUD"),
					resource.TestCheckResourceAttr(resourceName, "autoscaling_status", "INACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "allowed_usage", "1"),
					resource.TestCheckResourceAttr(resourceName, "scaling_strategy", "DEFAULT"),
					resource.TestCheckResourceAttr(resourceName, "auto_assignment", "true"),
					resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.cud_id", cudID),
					resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.plan", "TWELVE_MONTH"),
					resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.cpu", "32"),
					resource.TestCheckResourceAttrSet(resourceName, "state"),
					resource.TestCheckResourceAttrSet(resourceName, "create_time"),
				),
			},
			{
				// Patch-path update: operational settings only.
				Config: testAccCommitmentConfig(rName, cudID, "ACTIVE", 0.75, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "autoscaling_status", "ACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "allowed_usage", "0.75"),
				),
			},
			{
				// Upsert-path update: rename. ID must be preserved.
				Config: testAccCommitmentConfig(rName+"-renamed", cudID, "ACTIVE", 0.75, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-renamed"),
				),
			},
			{
				// Switch to explicit auto_assignment=false.
				Config: testAccCommitmentConfig(rName+"-renamed", cudID, "ACTIVE", 0.75, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
				),
			},
			{
				// Upsert-path update with explicit false: auto_assignment must stay false.
				Config: testAccCommitmentConfig(rName+"-renamed-2", cudID, "ACTIVE", 0.75, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-renamed-2"),
					resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource %s not found in state", resourceName)
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["organization_id"], rs.Primary.ID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCommitmentConfig(name, cudID, autoscalingStatus string, allowedUsage float64, setAutoAssignment bool, autoAssignment bool) string {
	autoAssignmentLine := ""
	if setAutoAssignment {
		autoAssignmentLine = fmt.Sprintf("auto_assignment = %v", autoAssignment)
	}
	return fmt.Sprintf(`
resource "castai_commitment" "test" {
  name               = %[1]q
  cloud              = "GCP"
  region             = "us-central1"
  type               = "RESOURCE_CUD"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = %[3]q
  allowed_usage      = %[4]v
  %[5]s

  gcp_resource_cud_details = {
    cud_id    = %[2]q
    plan      = "TWELVE_MONTH"
    type      = "GENERAL_PURPOSE_E2"
    cpu       = 32
    memory_mb = 131072
    status    = "ACTIVE"
  }
}
`, name, cudID, autoscalingStatus, allowedUsage, autoAssignmentLine)
}
