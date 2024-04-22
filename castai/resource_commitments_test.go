package castai

import (
	"testing"

	"github.com/castai/terraform-provider-castai/castai/commitments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestCommitments_GCP_BasicCUDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeGCPInitialCUDConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_id", "123456789"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_status", "ACTIVE"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.start_timestamp", "2023-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.end_timestamp", "2024-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.name", "test"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.region", "us-east4"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cpu", "10"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.memory_mb", "20480"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.plan", "TWELVE_MONTH"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.type", "COMPUTE_OPTIMIZED_C2D"),
				),
			},
			{
				ResourceName:            "castai_commitments.test_gcp",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{commitments.FieldGCPCUDsJSON},
			},
		},
	})
}

func makeGCPInitialCUDConfig() string {
	return `
resource "castai_commitments" "test_gcp" {
	gcp_cuds_json = <<EOF
[
  {
    "autoRenew": false,
    "category": "MACHINE",
    "creationTimestamp": "2023-01-01T00:00:00Z",
    "description": "",
    "endTimestamp": "2024-01-01T00:00:00Z",
    "id": "123456789",
    "kind": "compute#commitment",
    "name": "test",
    "plan": "TWELVE_MONTH",
    "region": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-east4",
    "resources": [
      {
        "amount": "10",
        "type": "VCPU"
      },
      {
        "amount": "20480",
        "type": "MEMORY"
      }
    ],
    "selfLink": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-east4/commitments/test",
    "startTimestamp": "2023-01-01T00:00:00Z",
    "status": "ACTIVE",
    "statusMessage": "The commitment is active, and so will apply to current resource usage.",
    "type": "COMPUTE_OPTIMIZED_C2D"
  }
]
	EOF
}
`
}
