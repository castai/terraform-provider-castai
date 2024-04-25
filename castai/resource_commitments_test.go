package castai

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/commitments"
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func TestCommitments_GCP_BasicCUDs(t *testing.T) {
	checkAttr := func(path, val string) resource.TestCheckFunc {
		return resource.TestCheckResourceAttr("castai_commitments.test_gcp", path, val)
	}
	// checkFloatAttr is a helper function to check float attributes with a precision of 3 decimal places.
	// The attributes map is a map[string]string, so floats in there may be affected by the rounding errors.
	checkFloatAttr := func(path string, val float64) func(state *terraform.State) error {
		return func(state *terraform.State) error {
			res, ok := state.RootModule().Resources["castai_commitments.test_gcp"]
			if !ok {
				return errors.New("resource not found")
			}
			v, ok := res.Primary.Attributes[path]
			if !ok {
				return errors.New("attribute not found")
			}
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			parsed = math.Round(parsed*1000) / 1000
			if parsed != val {
				return fmt.Errorf("expected %f, got %f", val, parsed)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			res, err := getOrganizationCommitments(ctx, testAccProvider.Meta())
			if err != nil {
				return err
			}
			res = lo.Filter(res, func(c sdk.CastaiInventoryV1beta1Commitment, _ int) bool {
				return c.GcpResourceCudContext != nil
			})
			if len(res) > 0 {
				return errors.New("gcp commitments still exist")
			}
			return nil
		},
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
					checkFloatAttr("gcp_cuds.0.allowed_usage", 1),
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
					checkFloatAttr("gcp_cuds.1.allowed_usage", 0.7),
					checkAttr("gcp_cuds.1.status", "Active"),
				),
			},
		},
	})
}

func TestCommitments_Azure_BasicReservations(t *testing.T) {
	checkAttr := func(path, val string) resource.TestCheckFunc {
		return resource.TestCheckResourceAttr("castai_commitments.test_azure", path, val)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		//CheckDestroy: TODO
		Steps: []resource.TestStep{
			{
				Config: initialAzureConfig,
				Check: resource.ComposeTestCheckFunc(
					checkAttr("azure_reservations.#", "1"),
					checkAttr("azure_reservations.0.reservation_id", "3b3de39c-bc44-4d69-be2d-69527dfe9958"),
					checkAttr("azure_reservations.0.reservation_status", "Succeeded"),
					checkAttr("azure_reservations.0.start_timestamp", "2023-01-11T00:00:00Z"),
					checkAttr("azure_reservations.0.end_timestamp", "2050-01-01T00:00:00Z"),
					checkAttr("azure_reservations.0.name", "test-res-1"),
					checkAttr("azure_reservations.0.region", "eastus"),
					checkAttr("azure_reservations.0.plan", "THREE_YEAR"),
					checkAttr("azure_reservations.0.instance_type", "Standard_D32as_v4"),
					checkAttr("azure_reservations.0.count", "3"),
					checkAttr("azure_reservations.0.scope", "Single subscription"),
					checkAttr("azure_reservations.0.scope_subscription", "8faa0959-093b-4612-8686-a996ac19db00"),
					checkAttr("azure_reservations.0.scope_resource_group", "All resource groups"),
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

	initialAzureConfig = `
resource "castai_commitments" "test_azure" {
	azure_reservations_csv = <<EOF
Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test-res-1,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
	EOF
}
`

	reservationsCSV = `Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test-res-1,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
test-res-2,3b3de39c-bc44-4d69-be2d-69527dfe9957,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:01Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,2,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview
test-res-3,3b3de39c-bc44-4d69-be2d-69527dfe9956,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:02Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,1,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/1745741b-f3c6-46a9-ad16-b93775a1bc38/overview`
)

type fakeReservationsProvider struct{}

func (p *fakeReservationsProvider) GetOk(_ string) (any, bool) {
	return reservationsCSV, true
}

func Test_getReservationImports(t *testing.T) {
	r := require.New(t)
	imports, ok, err := getReservationImports(&fakeReservationsProvider{})
	r.True(ok)
	r.NoError(err)
	r.Len(imports, 3)

	fmt.Println("----------")
	for _, imp := range imports {
		fmt.Printf("%+v\n", imp)
	}
	fmt.Println("----------")
}
