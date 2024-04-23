package castai

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/castai/terraform-provider-castai/castai/commitments"
)

func TestCommitments_GCP_BasicCUDs(t *testing.T) {
	checkAttr := func(path, val string) resource.TestCheckFunc {
		return resource.TestCheckResourceAttr("castai_commitments.test_gcp", path, val)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: initialGCPConfig,
				Check: resource.ComposeTestCheckFunc(
					checkAttr("gcp_cuds.#", "1"),
					checkAttr("gcp_cuds.0.cud_id", "123456789"),
					checkAttr("gcp_cuds.0.cud_status", "ACTIVE"),
					checkAttr("gcp_cuds.0.start_timestamp", "2023-01-01T00:00:00Z"),
					checkAttr("gcp_cuds.0.end_timestamp", "2024-01-01T00:00:00Z"),
					checkAttr("gcp_cuds.0.name", "test"),
					checkAttr("gcp_cuds.0.region", "us-east4"),
					checkAttr("gcp_cuds.0.cpu", "10"),
					checkAttr("gcp_cuds.0.memory_mb", "20480"),
					checkAttr("gcp_cuds.0.plan", "TWELVE_MONTH"),
					checkAttr("gcp_cuds.0.type", "COMPUTE_OPTIMIZED_C2D"),
				),
			},
			{
				ResourceName:            "castai_commitments.test_gcp",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{commitments.FieldGCPCUDsJSON},
			},
			{
				Config: updatedGCPConfig,
				Check: resource.ComposeTestCheckFunc(
					checkAttr("gcp_cuds.#", "2"),
					// "test" CUD
					checkAttr("gcp_cuds.0.cud_id", "123456789"),
					checkAttr("gcp_cuds.0.cud_status", "ACTIVE"),
					checkAttr("gcp_cuds.0.start_timestamp", "2023-01-01T00:00:00Z"),
					checkAttr("gcp_cuds.0.end_timestamp", "2024-01-01T00:00:00Z"),
					checkAttr("gcp_cuds.0.name", "test"),
					checkAttr("gcp_cuds.0.region", "us-east4"),
					checkAttr("gcp_cuds.0.cpu", "10"),
					checkAttr("gcp_cuds.0.memory_mb", "20480"),
					checkAttr("gcp_cuds.0.plan", "TWELVE_MONTH"),
					checkAttr("gcp_cuds.0.type", "COMPUTE_OPTIMIZED_C2D"),
					checkAttr("gcp_cuds.0.prioritization", "false"),
					//checkAttr("gcp_cuds.0.allowed_usage", "1"),
					checkAttr("gcp_cuds.0.status", "Active"),
					// "test-2" CUD
					checkAttr("gcp_cuds.1.cud_id", "987654321"),
					checkAttr("gcp_cuds.1.cud_status", "ACTIVE"),
					checkAttr("gcp_cuds.1.start_timestamp", "2023-06-01T00:00:00Z"),
					checkAttr("gcp_cuds.1.end_timestamp", "2024-06-01T00:00:00Z"),
					checkAttr("gcp_cuds.1.name", "test-2"),
					checkAttr("gcp_cuds.1.region", "us-east4"),
					checkAttr("gcp_cuds.1.cpu", "5"),
					checkAttr("gcp_cuds.1.memory_mb", "10240"),
					checkAttr("gcp_cuds.1.plan", "TWELVE_MONTH"),
					checkAttr("gcp_cuds.1.type", "GENERAL_PURPOSE_E2"),
					checkAttr("gcp_cuds.1.prioritization", "true"),
					//checkAttr("gcp_cuds.1.allowed_usage", "0.7"),
					checkAttr("gcp_cuds.1.status", "Active"),
				),
			},
		},
	})
}

var (
	initialGCPConfig = `
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

	updatedGCPConfig = `
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
  },
{
    "autoRenew": false,
    "category": "MACHINE",
    "creationTimestamp": "2023-06-01T00:00:00Z",
    "description": "",
    "endTimestamp": "2024-06-01T00:00:00Z",
    "id": "987654321",
    "kind": "compute#commitment",
    "name": "test-2",
    "plan": "TWELVE_MONTH",
    "region": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-east4",
    "resources": [
      {
        "amount": "5",
        "type": "VCPU"
      },
      {
        "amount": "10240",
        "type": "MEMORY"
      }
    ],
    "selfLink": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-east4/commitments/test-2",
    "startTimestamp": "2023-06-01T00:00:00Z",
    "status": "ACTIVE",
    "statusMessage": "The commitment is active, and so will apply to current resource usage.",
    "type": "GENERAL_PURPOSE_E2"
  }
]
	EOF

  gcp_cud_configs = [
    {
      matcher = {
        name = "test-2"
        type = "GENERAL_PURPOSE_E2"
        region = "us-east4"
      }
      prioritization = true
	  allowed_usage = 0.7
	  status = "Active"
    },
    {
      matcher = {
 	    name = "test"
        type = "COMPUTE_OPTIMIZED_C2D"
        region = "us-east4"
      }
      prioritization = false
      allowed_usage = 1
      status = "Active"
    }
  ]
}
`
)
