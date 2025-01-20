package castai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestAccCommitments(t *testing.T) {
	var (
		gcpServiceAccountID = fmt.Sprintf("%v-node-cfg-%v", ResourcePrefix, acctest.RandString(8))
		gkeClusterName      = "tf-core-acc-20230723"
		gcpProjectID        = os.Getenv("GOOGLE_PROJECT_ID")

		azureRoleName              = fmt.Sprintf("%v-aks-%v", ResourcePrefix, acctest.RandString(8))
		azureClusterName           = "core-tf-acc"
		azureResourceGroupName     = "core-tf-acc"
		azureNodeResourceGroupName = "core-tf-acc-ng"
	)

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
			if len(res) > 0 {
				return errors.New("commitments still exist")
			}
			return nil
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
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 3.7.0",
			},
			"azuread": {
				Source:            "hashicorp/azuread",
				VersionConstraint: "~> 2.22.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: getCommitmentsConfig(gcpServiceAccountID, gkeClusterName, gcpProjectID, azureRoleName, azureClusterName, azureResourceGroupName, azureNodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "2"),
					// GCP - "test" CUD
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_id", "123456789"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_status", "ACTIVE"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.start_timestamp", "2023-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.end_timestamp", "2024-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.name", "test"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.region", "us-central1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cpu", "10"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.memory_mb", "20480"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.plan", "TWELVE_MONTH"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.type", "COMPUTE_OPTIMIZED_C2D"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.prioritization", "false"),
					checkFloatAttr("castai_commitments.test_gcp", "gcp_cuds.0.allowed_usage", 1),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.status", "Active"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.assignments.#", "1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.scaling_strategy", "Default"),
					// GCP - "test-2" CUD
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.cud_id", "987654321"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.cud_status", "ACTIVE"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.start_timestamp", "2023-06-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.end_timestamp", "2024-06-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.name", "test-2"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.region", "us-central1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.cpu", "5"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.memory_mb", "10240"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.plan", "TWELVE_MONTH"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.type", "GENERAL_PURPOSE_E2"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.prioritization", "true"),
					checkFloatAttr("castai_commitments.test_gcp", "gcp_cuds.1.allowed_usage", 0.7),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.status", "Active"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.assignments.#", "1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.1.scaling_strategy", "CPUBased"),

					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.#", "2"),
					// Azure - "test-res-1" RI
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.reservation_id", "3b3de39c-bc44-4d69-be2d-69527dfe9958"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.reservation_status", "Succeeded"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.start_timestamp", "2023-01-11T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.end_timestamp", "2050-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.name", "test-res-1"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.region", "westeurope"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.plan", "THREE_YEAR"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.instance_type", "Standard_D32as_v4"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.count", "3"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.scope", "Single subscription"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.scope_subscription", "8faa0959-093b-4612-8686-a996ac19db00"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.scope_resource_group", "All resource groups"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.prioritization", "false"),
					checkFloatAttr("castai_commitments.test_azure", "azure_reservations.0.allowed_usage", 0.6),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.status", "Active"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.0.assignments.#", "1"),
					// Azure - "test-res-2" RI
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.reservation_id", "3b3de39c-bc44-4d69-be2d-69527dfe9959"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.reservation_status", "Succeeded"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.start_timestamp", "2023-01-12T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.end_timestamp", "2040-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.name", "test-res-2"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.region", "westeurope"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.plan", "ONE_YEAR"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.instance_type", "Standard_B1s"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.count", "2"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.scope", "Single subscription"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.scope_subscription", "8faa0959-093b-4612-8686-a996ac19db00"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.scope_resource_group", "All resource groups"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.prioritization", "true"),
					checkFloatAttr("castai_commitments.test_azure", "azure_reservations.1.allowed_usage", 0.9),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.status", "Active"),
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.1.assignments.#", "1"),
				),
			},
			{
				ResourceName:            "castai_commitments.test_gcp",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{fieldCommitmentsGCPCUDsJSON, fieldCommitmentsConfigs},
			},
			{
				ResourceName:            "castai_commitments.test_azure",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{fieldCommitmentsAzureReservationsCSV, fieldCommitmentsConfigs},
			},
		},
	})
}

var (
	commitment1Obj = `{
    "autoRenew": false,
    "category": "MACHINE",
    "creationTimestamp": "2023-01-01T00:00:00Z",
    "description": "",
    "endTimestamp": "2024-01-01T00:00:00Z",
    "id": "123456789",
    "kind": "compute#commitment",
    "name": "test",
    "plan": "TWELVE_MONTH",
    "region": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1",
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
    "selfLink": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/commitments/test",
    "startTimestamp": "2023-01-01T00:00:00Z",
    "status": "ACTIVE",
    "statusMessage": "The commitment is active, and so will apply to current resource usage.",
    "type": "COMPUTE_OPTIMIZED_C2D"
  }`

	commitment2Obj = `{
    "autoRenew": false,
    "category": "MACHINE",
    "creationTimestamp": "2023-06-01T00:00:00Z",
    "description": "",
    "endTimestamp": "2024-06-01T00:00:00Z",
    "id": "987654321",
    "kind": "compute#commitment",
    "name": "test-2",
    "plan": "TWELVE_MONTH",
    "region": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1",
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
    "selfLink": "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/commitments/test-2",
    "startTimestamp": "2023-06-01T00:00:00Z",
    "status": "ACTIVE",
    "statusMessage": "The commitment is active, and so will apply to current resource usage.",
    "type": "GENERAL_PURPOSE_E2"
  }`
)

func getCommitmentsConfig(
	gcpServiceAccountID, gcpClusterName, gcpProjectID,
	azureRoleName, azureClusterName, azureResourceGroupName, azureNodeResourceGroupName string,
) string {
	return ConfigCompose(
		testAccGKEClusterConfig(gcpServiceAccountID, gcpClusterName, gcpProjectID),
		testAccAKSClusterConfig(azureRoleName, azureClusterName, azureResourceGroupName, azureNodeResourceGroupName),
		`
resource "castai_commitments" "test_gcp" {
	gcp_cuds_json = <<EOF
[
   `+commitment1Obj+`,
   `+commitment2Obj+`
]
	EOF

	commitment_configs {
		matcher {
 			name = "test"
			type = "COMPUTE_OPTIMIZED_C2D"
			region = "us-central1"
		}
		prioritization = false
		allowed_usage = 1
		status = "Active"
		assignments {
			cluster_id = castai_gke_cluster.test.id
	  	}
		scaling_strategy = "Default"
	}

  	commitment_configs {
		matcher {
			name = "test-2"
			type = "GENERAL_PURPOSE_E2"
			region = "us-central1"
		}	
		prioritization = true
		allowed_usage = 0.7
		status = "Active"
		assignments {
			cluster_id = castai_gke_cluster.test.id
		}
		scaling_strategy = "CPUBased"
   }
}

resource "castai_commitments" "test_azure" {
	azure_reservations_csv = <<EOF
Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test-res-1,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,westeurope,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
test-res-2,3b3de39c-bc44-4d69-be2d-69527dfe9959,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2040-01-01T00:00:00Z,2023-01-12T00:00:00Z,P1Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_B1s,westeurope,2,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
	EOF

	commitment_configs {
		matcher {
			name = "test-res-1"
			region = "westeurope"
			type = "Standard_D32as_v4"
		}
		prioritization = false
		allowed_usage = 0.6
		status = "Active"
		assignments {
			cluster_id = castai_aks_cluster.test.id
		}
		scaling_strategy = "Default"
	}

	commitment_configs {
		matcher {
			name = "test-res-2"
			region = "westeurope"
			type = "Standard_B1s"
		}
		prioritization = true
		allowed_usage = 0.9
		status = "Active"
		assignments {
			cluster_id = castai_aks_cluster.test.id
		}
		scaling_strategy = "Default"
	}
}
`)
}

// Both create and update use the same "upsert" handler under the hood so we test them together
func TestCommitmentsResourceCreateAndUpdate(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	orgID, clusterID, commitmentID := uuid.New(), uuid.New(), uuid.New()

	type test struct {
		resource               map[string]any
		commitmentImport       any // CastaiInventoryV1beta1GCPCommitmentImport | CastaiInventoryV1beta1AzureReservationImport
		expectCommitmentUpdate sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody
		mockImportedCommitment sdk.CastaiInventoryV1beta1Commitment
	}

	gcpImport := sdk.CastaiInventoryV1beta1GCPCommitmentImport{
		AutoRenew:         lo.ToPtr(true),
		Category:          lo.ToPtr("MACHINE"),
		CreationTimestamp: lo.ToPtr("2023-01-01T00:00:00Z"),
		Description:       lo.ToPtr("some description"),
		EndTimestamp:      lo.ToPtr("2024-01-01T00:00:00Z"),
		Id:                lo.ToPtr("123456789"),
		Kind:              lo.ToPtr("compute#commitment"),
		Name:              lo.ToPtr("test"),
		Plan:              lo.ToPtr("TWELVE_MONTH"),
		Region:            lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1"),
		Resources: &[]sdk.CastaiInventoryV1beta1GCPResource{
			{Amount: lo.ToPtr("10"), Type: lo.ToPtr("VCPU")},
			{Amount: lo.ToPtr("20480"), Type: lo.ToPtr("MEMORY")},
		},
		SelfLink:       lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/commitments/test"),
		StartTimestamp: lo.ToPtr("2023-01-01T00:00:00Z"),
		Status:         lo.ToPtr("ACTIVE"),
		StatusMessage:  lo.ToPtr("The commitment is active, and so will apply to current resource usage."),
		Type:           lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
	}

	// Note that the import endpoints are called in "overwrite" mode. This means that we don't need to test scenarios
	// where certain commitments already exist and some of them need to be deleted because they don't exist anymore
	// in the new plan.
	tests := map[string]test{
		"should create a gcp commitment": func() test {
			return test{
				resource: map[string]any{
					fieldCommitmentsGCPCUDsJSON: toJSONString(r, []sdk.CastaiInventoryV1beta1GCPCommitmentImport{gcpImport}),
					fieldCommitmentsConfigs: []any{
						map[string]any{
							"matcher": []any{
								map[string]any{
									"name":   "test",
									"type":   "COMPUTE_OPTIMIZED_C2D",
									"region": "us-central1",
								},
							},
							"assignments": []any{
								map[string]any{
									"cluster_id": clusterID.String(),
									"priority":   1,
								},
							},
							"prioritization":   true,
							"status":           "Active",
							"allowed_usage":    0.6,
							"scaling_strategy": "CPUBased",
						},
					},
				},
				commitmentImport: gcpImport,
				expectCommitmentUpdate: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
					AllowedUsage:    lo.ToPtr[float32](0.6),
					Prioritization:  lo.ToPtr(true),
					ScalingStrategy: lo.ToPtr(sdk.CPUBased),
					Status:          lo.ToPtr(sdk.Active),
				},
				mockImportedCommitment: sdk.CastaiInventoryV1beta1Commitment{
					EndDate: lo.ToPtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					Id:      lo.ToPtr(commitmentID.String()),
					Name:    lo.ToPtr("test"),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{
						Cpu:      lo.ToPtr("10"),
						CudId:    lo.ToPtr("123456789"),
						MemoryMb: lo.ToPtr("20480"),
						Plan:     lo.ToPtr(sdk.TWELVEMONTH),
						Status:   lo.ToPtr("Active"),
						Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
					},
					Region:       lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1"),
					StartDate:    lo.ToPtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Status:       lo.ToPtr(sdk.Active),
					AllowedUsage: lo.ToPtr[float32](1),
				},
			}
		}(),
		"should create an azure commitment": func() test {
			commitmentImport := sdk.CastaiInventoryV1beta1AzureReservationImport{
				ExpirationDate:     lo.ToPtr("2050-01-01T00:00:00Z"),
				Name:               lo.ToPtr("test"),
				ProductName:        lo.ToPtr("Standard_D32as_v4"),
				PurchaseDate:       lo.ToPtr("2023-01-11T00:00:00Z"),
				Quantity:           lo.ToPtr[int32](3),
				Region:             lo.ToPtr("eastus"),
				ReservationId:      lo.ToPtr("3b3de39c-bc44-4d69-be2d-69527dfe9958"),
				Scope:              lo.ToPtr("Single subscription"),
				ScopeResourceGroup: lo.ToPtr("All resource groups"),
				ScopeSubscription:  lo.ToPtr("8faa0959-093b-4612-8686-a996ac19db00"),
				Status:             lo.ToPtr("Succeeded"),
				Term:               lo.ToPtr("P3Y"),
				Type:               lo.ToPtr("VirtualMachines"),
			}

			return test{
				resource: map[string]any{
					fieldCommitmentsAzureReservationsCSV: `Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview`,
					fieldCommitmentsConfigs: []any{
						map[string]any{
							"matcher": []any{
								map[string]any{
									"name":   "test",
									"type":   "Standard_D32as_v4",
									"region": "eastus",
								},
							},
							"assignments": []any{
								map[string]any{
									"cluster_id": clusterID.String(),
									"priority":   1,
								},
							},
							"prioritization":   true,
							"status":           "Active",
							"allowed_usage":    0.7,
							"scaling_strategy": "Default",
						},
					},
				},
				commitmentImport: commitmentImport,
				expectCommitmentUpdate: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
					AllowedUsage:    lo.ToPtr[float32](0.7),
					Prioritization:  lo.ToPtr(true),
					ScalingStrategy: lo.ToPtr(sdk.Default),
					Status:          lo.ToPtr(sdk.Active),
				},
				mockImportedCommitment: sdk.CastaiInventoryV1beta1Commitment{
					EndDate: lo.ToPtr(time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)),
					Id:      lo.ToPtr(commitmentID.String()),
					Name:    lo.ToPtr("test"),
					Region:  lo.ToPtr("eastus"),
					AzureReservationContext: &sdk.CastaiInventoryV1beta1AzureReservation{
						Count:                 lo.ToPtr[int32](3),
						Id:                    lo.ToPtr("3b3de39c-bc44-4d69-be2d-69527dfe9958"),
						InstanceType:          lo.ToPtr("Standard_D32as_v4"),
						InstanceTypeCpu:       lo.ToPtr("32"),
						InstanceTypeMemoryMib: lo.ToPtr("131072"),
						Plan:                  lo.ToPtr(sdk.THREEYEAR),
						Scope:                 lo.ToPtr("Single subscription"),
						ScopeResourceGroup:    lo.ToPtr("All resource groups"),
						ScopeSubscription:     lo.ToPtr("8faa0959-093b-4612-8686-a996ac19db00"),
						Status:                lo.ToPtr("Succeeded"),
					},
				},
			}
		}(),
	}

	type testedFn string
	const (
		testedFnCreate testedFn = "create"
		testedFnUpdate testedFn = "update"
	)
	for _, testedFn := range []testedFn{testedFnCreate, testedFnUpdate} {
		t.Run(string(testedFn), func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					r := require.New(t)

					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					resource := resourceCommitments()
					mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
					provider := &ProviderConfig{api: mockClient}

					// Fetches the default organization ID
					mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
						JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
							Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
								{Id: lo.ToPtr(orgID.String())}, // the first org is the default one so everything else should be ignored
								{Id: lo.ToPtr(uuid.New().String())},
							},
						},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).Times(1)

					data := schema.TestResourceDataRaw(t, resource.Schema, tt.resource)

					// Actual commitments import call
					switch v := tt.commitmentImport.(type) {
					case sdk.CastaiInventoryV1beta1GCPCommitmentImport:
						mockClient.EXPECT().CommitmentsAPIImportGCPCommitmentsWithResponse(
							gomock.Any(),
							&sdk.CommitmentsAPIImportGCPCommitmentsParams{
								Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportGCPCommitmentsParamsBehaviour]("OVERWRITE"),
							},
							[]sdk.CastaiInventoryV1beta1GCPCommitmentImport{v},
						).Return(&sdk.CommitmentsAPIImportGCPCommitmentsResponse{
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)
					case sdk.CastaiInventoryV1beta1AzureReservationImport:
						mockClient.EXPECT().CommitmentsAPIImportAzureReservationsWithResponse(
							gomock.Any(),
							&sdk.CommitmentsAPIImportAzureReservationsParams{
								Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportAzureReservationsParamsBehaviour]("OVERWRITE"),
							},
							[]sdk.CastaiInventoryV1beta1AzureReservationImport{v},
						).Return(&sdk.CommitmentsAPIImportAzureReservationsResponse{
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)
					}

					// There are 2 get commitments calls: one during the creation and one by the state importer
					mockClient.EXPECT().CommitmentsAPIGetCommitmentsWithResponse(
						gomock.Any(), &sdk.CommitmentsAPIGetCommitmentsParams{},
					).Return(&sdk.CommitmentsAPIGetCommitmentsResponse{
						JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsResponse{
							Commitments: &[]sdk.CastaiInventoryV1beta1Commitment{tt.mockImportedCommitment},
						},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).Times(2)

					// Update is called after importing the commitments to set fields such as status,
					// allowed usage, etc. specified in the config
					mockClient.EXPECT().CommitmentsAPIUpdateCommitmentWithResponse(
						gomock.Any(), commitmentID.String(), tt.expectCommitmentUpdate,
					).Return(&sdk.CommitmentsAPIUpdateCommitmentResponse{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200:      &sdk.CastaiInventoryV1beta1UpdateCommitmentResponse{},
					}, nil).Times(1)

					// Assignments replace is called to assign the commitment to clusters specified in the config
					mockClient.EXPECT().CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(
						gomock.Any(),
						commitmentID.String(),
						sdk.CommitmentsAPIReplaceCommitmentAssignmentsJSONRequestBody{clusterID.String()},
					).Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200:      &sdk.CastaiInventoryV1beta1ReplaceCommitmentAssignmentsResponse{},
					}, nil).Times(1)

					// Commitment assignments are fetched by the state importer
					mockClient.EXPECT().CommitmentsAPIGetCommitmentsAssignmentsWithResponse(gomock.Any()).
						Return(&sdk.CommitmentsAPIGetCommitmentsAssignmentsResponse{
							JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsAssignmentsResponse{
								CommitmentsAssignments: &[]sdk.CastaiInventoryV1beta1CommitmentAssignment{},
							},
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)

					var fn func(context.Context, *schema.ResourceData, any) diag.Diagnostics
					switch testedFn {
					case testedFnCreate:
						fn = resource.CreateContext
					case testedFnUpdate:
						fn = resource.UpdateContext
					default:
						r.Failf("unexpected tested function: %s", string(testedFn))
					}

					diag := fn(ctx, data, provider)
					noErrInDiagnostics(r, diag)
				})
			}
		})
	}
}

func TestCommitmentsResourceRead(t *testing.T) {
	ctx := context.Background()
	orgID, clusterID, commitment1ID, commitment2ID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tests := map[string]struct {
		stateKey    string
		id          string
		commitments []sdk.CastaiInventoryV1beta1Commitment
		assignments []sdk.CastaiInventoryV1beta1CommitmentAssignment
		checkState  func(*require.Assertions, any)
	}{
		"should import gcp commitments with assignments": {
			stateKey: fieldCommitmentsGCPCUDs,
			id:       orgID.String() + ":gcp",
			// Mapper functions are tested in their own unit tests, hence we only do basic checks here
			commitments: []sdk.CastaiInventoryV1beta1Commitment{
				{
					Id:                    lo.ToPtr(commitment1ID.String()),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{},
				},
				{
					Id:                    lo.ToPtr(commitment2ID.String()),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{},
				},
			},
			assignments: []sdk.CastaiInventoryV1beta1CommitmentAssignment{
				{
					ClusterId:    lo.ToPtr(clusterID.String()),
					CommitmentId: lo.ToPtr(commitment1ID.String()),
				},
				{
					ClusterId:    lo.ToPtr(clusterID.String()),
					CommitmentId: lo.ToPtr(commitment2ID.String()),
				},
			},
			checkState: func(r *require.Assertions, v any) {
				var parsed []gcpCUDResource
				r.NoError(mapstructure.Decode(v, &parsed))
				r.Len(parsed, 2)

				r.Equal(commitment1ID.String(), parsed[0].getCommitmentID())
				r.Len(parsed[0].Assignments, 1)
				r.Equal(clusterID.String(), parsed[0].Assignments[0].ClusterID)

				r.Equal(commitment2ID.String(), parsed[1].getCommitmentID())
				r.Len(parsed[1].Assignments, 1)
				r.Equal(clusterID.String(), parsed[1].Assignments[0].ClusterID)
			},
		},
		"should import azure commitments with no assignments": {
			stateKey: fieldCommitmentsAzureReservations,
			id:       orgID.String() + ":azure",
			commitments: []sdk.CastaiInventoryV1beta1Commitment{
				{
					Id:                      lo.ToPtr(commitment1ID.String()),
					AzureReservationContext: &sdk.CastaiInventoryV1beta1AzureReservation{},
				},
			},
			checkState: func(r *require.Assertions, v any) {
				var parsed []azureReservationResource
				r.NoError(mapstructure.Decode(v, &parsed))
				r.Len(parsed, 1)
				r.Equal(commitment1ID.String(), parsed[0].getCommitmentID())
				r.Len(parsed[0].Assignments, 0)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := resourceCommitments()

			mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
			provider := &ProviderConfig{api: mockClient}

			data := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{tt.stateKey: []any{}})
			data.SetId(tt.id)

			mockClient.EXPECT().
				CommitmentsAPIGetCommitmentsWithResponse(gomock.Any(), &sdk.CommitmentsAPIGetCommitmentsParams{}).
				Return(&sdk.CommitmentsAPIGetCommitmentsResponse{
					JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsResponse{
						Commitments: &tt.commitments,
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).
				Times(1)

			mockClient.EXPECT().
				CommitmentsAPIGetCommitmentsAssignmentsWithResponse(gomock.Any()).
				Return(&sdk.CommitmentsAPIGetCommitmentsAssignmentsResponse{
					JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsAssignmentsResponse{
						CommitmentsAssignments: &tt.assignments,
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).
				Times(1)

			diag := resource.ReadContext(ctx, data, provider)
			noErrInDiagnostics(r, diag)

			v := data.Get(tt.stateKey)
			if tt.checkState != nil {
				tt.checkState(r, v)
			}
		})
	}
}

func TestCommitmentsResourceDelete(t *testing.T) {
	ctx := context.Background()
	orgID, commitmentID := uuid.New(), uuid.New()

	tests := map[string]struct {
		resource     map[string]any
		expectDelete bool
	}{
		"should be a no-op when no commitments are present": {
			resource: map[string]any{},
		},
		"should delete gcp commitments resource": {
			resource: map[string]any{
				fieldCommitmentsGCPCUDs: []any{
					map[string]any{
						"id": commitmentID.String(),
					},
				},
			},
			expectDelete: true,
		},
		"should delete azure commitments resource": {
			resource: map[string]any{
				fieldCommitmentsAzureReservations: []any{
					map[string]any{
						"id": commitmentID.String(),
					},
				},
			},
			expectDelete: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := resourceCommitments()
			mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
			provider := &ProviderConfig{api: mockClient}

			data := schema.TestResourceDataRaw(t, resource.Schema, tt.resource)

			// Fetches the default organization ID to get commitments import ID
			mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
				JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
					Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
						{Id: lo.ToPtr(orgID.String())}, // the first org is the default one so everything else should be ignored
						{Id: lo.ToPtr(uuid.New().String())},
					},
				},
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).Times(1)

			if tt.expectDelete {
				mockClient.EXPECT().
					CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), commitmentID.String()).
					Return(&sdk.CommitmentsAPIDeleteCommitmentResponse{
						JSON200:      &map[string]any{},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).
					Times(1)
			}

			diag := resource.DeleteContext(ctx, data, provider)
			noErrInDiagnostics(r, diag)
		})
	}
}

func noErrInDiagnostics(r *require.Assertions, diags diag.Diagnostics) {
	for _, d := range diags {
		if d.Severity == diag.Error {
			r.Failf("unexpected error: %s", d.Summary)
		}
	}
}

func toJSONString(r *require.Assertions, v any) string {
	raw, err := json.Marshal(v)
	r.NoError(err)
	return string(raw)
}
