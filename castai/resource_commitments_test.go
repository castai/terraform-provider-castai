package castai

import (
	"testing"

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
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.name", "test"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.type", "COMPUTE_OPTIMIZED_C2D"),
				),
			},
			//{
			//	ResourceName:            "castai_reservations.test",
			//	ImportState:             true,
			//	ImportStateVerify:       true,
			//	ImportStateVerifyIgnore: []string{reservations.FieldReservationsCSV},
			//},
			//{
			//	Config: makeAzureUpdatedReservationsConfig(),
			//	Check: resource.ComposeTestCheckFunc(
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.#", "3"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.name", "VM_RI_01-01-2023_01-01"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.name", "VM_RI_01-01-2023_01-02"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.name", "VM_RI_01-01-2023_01-03"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.provider", "azure"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.provider", "azure"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.provider", "azure"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.count", "3"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.count", "2"),
			//		resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.count", "1"),
			//	),
			//},
		},
	})
}

func makeGCPInitialCUDConfig() string {
	return `
resource "castai_commitments" "test_gcp" {
	reservations_csv = <<EOF
[
  {
    "autoRenew": false,
    "category": "MACHINE",
    "creationTimestamp": "2023-01-01T00:00:00.000-07:00",
    "description": "",
    "endTimestamp": "2024-01-01T00:00:00.000-07:00",
    "id": "123456789",
    "kind": "compute#commitment",
    "name": "test",
    "plan": "TWELVE_MONTH",
    "region": "https://www.googleapis.com/compute/v1/projects/prod-master-scl0/regions/us-east4",
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
    "selfLink": "https://www.googleapis.com/compute/v1/projects/test/regions/us-east4/commitments/test",
    "startTimestamp": "2023-10-21T00:00:00.000-07:00",
    "status": "ACTIVE",
    "statusMessage": "The commitment is active, and so will apply to current resource usage.",
    "type": "COMPUTE_OPTIMIZED_C2D"
  }
]
	EOF
}
`
}

//func makeAzureUpdatedReservationsConfig() string {
//	return `
//resource "castai_reservations" "test" {
//	reservations_csv = <<EOF
//Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
//VM_RI_01-01-2023_01-01,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
//VM_RI_01-01-2023_01-02,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:01Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,2,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview
//VM_RI_01-01-2023_01-03,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:02Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,1,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/1745741b-f3c6-46a9-ad16-b93775a1bc38/overview
//	EOF
//}
//`
//}
