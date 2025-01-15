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

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestAccCommitments(t *testing.T) {
	// TODO: FIX by https://castai.atlassian.net/browse/PRICE-410
	t.Skip()
	// END TODO
	var (
		gcpServiceAccountID = fmt.Sprintf("%v-node-cfg-%v", ResourcePrefix, acctest.RandString(8))
		gkeClusterName      = "tf-core-acc-20230723"
		gcpProjectID        = os.Getenv("GOOGLE_PROJECT_ID")

		azureRoleName              = fmt.Sprintf("%v-aks-%v", ResourcePrefix, acctest.RandString(8))
		azureClusterName           = "core-tf-acc"
		azureResourceGroupName     = "core-tf-acc"
		azureNodeResourceGroupName = "core-tf-acc-ng"
	)

	importCUDsStateStep := resource.TestStep{
		ResourceName:            "castai_commitments.test_gcp",
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: []string{fieldCommitmentsGCPCUDsJSON, fieldCommitmentsConfigs},
	}
	importReservationsStateStep := resource.TestStep{
		ResourceName:            "castai_commitments.test_azure",
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: []string{fieldCommitmentsAzureReservationsCSV, fieldCommitmentsConfigs},
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
			{ // Import 2 commitments - one GCP CUD and one Azure reservations, both without configs
				Config: getCommitmentsConfig1(gcpServiceAccountID, gkeClusterName, gcpProjectID),
				Check: resource.ComposeTestCheckFunc(
					// GCP
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "1"),
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
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.assignments.#", "0"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.scaling_strategy", "Default"),
					// Azure
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.#", "1"),
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
				),
			},
			importCUDsStateStep,
			importReservationsStateStep,
			{ // Add config to the first GCP CUD, add another GCP CUD, Azure reservation remains unchanged
				Config: getCommitmentsConfig2(gcpServiceAccountID, gkeClusterName, gcpProjectID, azureRoleName, azureClusterName, azureResourceGroupName, azureNodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "2"),
					// GCP - "test" CUD, added config
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
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.assignments.#", "0"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.scaling_strategy", "Default"),
					// GCP - "test-2" CUD, added in the update with config
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
					// Azure - unchanged
					resource.TestCheckResourceAttr("castai_commitments.test_azure", "azure_reservations.#", "1"),
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
				),
			},
			importCUDsStateStep,
			importReservationsStateStep,
			{ // CUDs are unchanged, add config to the first Azure reservation and add another Azure reservation
				Config: getCommitmentsConfig3(gcpServiceAccountID, gkeClusterName, gcpProjectID, azureRoleName, azureClusterName, azureResourceGroupName, azureNodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "2"),
					// GCP - "test" CUD, unchanged
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
					// GCP - "test-2" CUD, unchanged
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
					// Azure - "test-res-1", added config
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
					// Azure - "test-res-2", added in the update with config
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
			importCUDsStateStep,
			importReservationsStateStep,
			{ // CUDs are unchanged, destroy the Azure import
				Config: getCommitmentsConfig4(gcpServiceAccountID, gkeClusterName, gcpProjectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "2"),
					// GCP - "test" CUD, unchanged
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
					// GCP - "test-2" CUD, unchanged
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
				),
			},
			importCUDsStateStep,
			{ // Remove the first GCP CUD so that the second one remains
				Config: getCommitmentsConfig5(gcpServiceAccountID, gkeClusterName, gcpProjectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.#", "1"),
					// GCP - "test-2" CUD, unchanged
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_id", "987654321"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cud_status", "ACTIVE"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.start_timestamp", "2023-06-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.end_timestamp", "2024-06-01T00:00:00Z"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.name", "test-2"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.region", "us-central1"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.cpu", "5"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.memory_mb", "10240"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.plan", "TWELVE_MONTH"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.type", "GENERAL_PURPOSE_E2"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.prioritization", "true"),
					checkFloatAttr("castai_commitments.test_gcp", "gcp_cuds.0.allowed_usage", 0.7),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.status", "Active"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.assignments.#", "0"),
					resource.TestCheckResourceAttr("castai_commitments.test_gcp", "gcp_cuds.0.scaling_strategy", "Default"),
				),
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

func getCommitmentsConfig1(serviceAccountID, clusterName, projectID string) string {
	return ConfigCompose(testAccGKEClusterConfig(serviceAccountID, clusterName, projectID), `
resource "castai_commitments" "test_gcp" {
	gcp_cuds_json = <<EOF
[
  `+commitment1Obj+`
]
	EOF
}

resource "castai_commitments" "test_azure" {
	azure_reservations_csv = <<EOF
Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test-res-1,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,westeurope,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
	EOF
}
`)
}

func getCommitmentsConfig2(
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
	EOF
}
`)
}

func getCommitmentsConfig3(
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

func getCommitmentsConfig4(serviceAccountID, clusterName, projectID string) string {
	return ConfigCompose(testAccGKEClusterConfig(serviceAccountID, clusterName, projectID), `
provider "azurerm" {
  features {}
}

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
`)
}

func getCommitmentsConfig5(serviceAccountID, clusterName, projectID string) string {
	return ConfigCompose(testAccGKEClusterConfig(serviceAccountID, clusterName, projectID), `
provider "azurerm" {
  features {}
}

resource "castai_commitments" "test_gcp" {
	gcp_cuds_json = <<EOF
[
    `+commitment2Obj+`
]
	EOF

  commitment_configs {
	matcher {
	  name = "test-2"
	  type = "GENERAL_PURPOSE_E2"
	  region = "us-central1"
	}
	prioritization = true
	allowed_usage = 0.7
	status = "Active"
	scaling_strategy = "Default"
  }
}
`)
}
func TestResourceCreate(t *testing.T) {
	ctx := context.Background()
	resource := resourceCommitments()
	r := require.New(t)

	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
	provider := &ProviderConfig{api: mockClient}

	orgID, clusterID, commitmentID := uuid.New(), uuid.New(), uuid.New()

	type test struct {
		resource               map[string]any
		commitmentImport       sdk.CastaiInventoryV1beta1GCPCommitmentImport
		expectCommitmentUpdate sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody
		mockImportedCommitment sdk.CastaiInventoryV1beta1Commitment
	}

	tests := map[string]test{
		"should create a gcp commitment": func() test {
			commitmentImport := sdk.CastaiInventoryV1beta1GCPCommitmentImport{
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

			return test{
				resource: map[string]any{
					fieldCommitmentsGCPCUDsJSON: func() string {
						raw, err := json.Marshal([]sdk.CastaiInventoryV1beta1GCPCommitmentImport{commitmentImport})
						r.NoError(err)
						return string(raw)
					}(),
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
				commitmentImport: commitmentImport,
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
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

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

			mockClient.EXPECT().CommitmentsAPIImportGCPCommitmentsWithResponse(
				gomock.Any(),
				&sdk.CommitmentsAPIImportGCPCommitmentsParams{
					Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportGCPCommitmentsParamsBehaviour]("OVERWRITE"),
				},
				[]sdk.CastaiInventoryV1beta1GCPCommitmentImport{tt.commitmentImport},
			).Return(&sdk.CommitmentsAPIImportGCPCommitmentsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).Times(1)

			mockClient.EXPECT().CommitmentsAPIGetCommitmentsWithResponse(
				gomock.Any(), &sdk.CommitmentsAPIGetCommitmentsParams{},
			).Return(&sdk.CommitmentsAPIGetCommitmentsResponse{
				JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsResponse{
					Commitments: &[]sdk.CastaiInventoryV1beta1Commitment{tt.mockImportedCommitment},
				},
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).Times(2)

			mockClient.EXPECT().CommitmentsAPIUpdateCommitmentWithResponse(
				gomock.Any(), commitmentID.String(), tt.expectCommitmentUpdate,
			).Return(&sdk.CommitmentsAPIUpdateCommitmentResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &sdk.CastaiInventoryV1beta1UpdateCommitmentResponse{},
			}, nil).Times(1)

			mockClient.EXPECT().CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(
				gomock.Any(),
				commitmentID.String(),
				sdk.CommitmentsAPIReplaceCommitmentAssignmentsJSONRequestBody{clusterID.String()},
			).Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &sdk.CastaiInventoryV1beta1ReplaceCommitmentAssignmentsResponse{},
			}, nil).Times(1)

			mockClient.EXPECT().CommitmentsAPIGetCommitmentsAssignmentsWithResponse(gomock.Any()).
				Return(&sdk.CommitmentsAPIGetCommitmentsAssignmentsResponse{
					JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsAssignmentsResponse{
						CommitmentsAssignments: &[]sdk.CastaiInventoryV1beta1CommitmentAssignment{},
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).Times(1)

			diag := resource.CreateContext(ctx, data, provider)
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
