package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/castai/terraform-provider-castai/castai/reservations"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestReservations_Azure_BasicReservationsCSV(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeAzureInitialReservationsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.#", "2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.name", "VM_RI_01-01-2023_01-01"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.name", "VM_RI_01-01-2023_01-02"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.provider", "azure"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.provider", "azure"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.count", "1"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.count", "2"),
				),
			},
			{
				ResourceName:            "castai_reservations.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{reservations.FieldReservationsCSV},
			},
			{
				Config: makeAzureUpdatedReservationsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.#", "3"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.name", "VM_RI_01-01-2023_01-01"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.name", "VM_RI_01-01-2023_01-02"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.name", "VM_RI_01-01-2023_01-03"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.provider", "azure"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.provider", "azure"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.provider", "azure"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.count", "3"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.count", "2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.count", "1"),
				),
			},
		},
	})
}

func TestReservations_Generic_BasicReservationsCSV(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeGenericInitialReservationsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.#", "2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.name", "reservation1"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.name", "reservation2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.provider", "aws"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.provider", "aws"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.count", "1"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.count", "2"),
				),
			},
			{
				ResourceName:            "castai_reservations.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{reservations.FieldReservationsCSV},
			},
			{
				Config: makeGenericUpdatedReservationsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.#", "3"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.name", "reservation1"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.name", "reservation2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.name", "reservation3"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.provider", "aws"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.provider", "aws"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.provider", "aws"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.0.count", "3"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.1.count", "2"),
					resource.TestCheckResourceAttr("castai_reservations.test", "reservations.2.count", "1"),
				),
			},
		},
	})
}

func makeAzureInitialReservationsConfig() string {
	return `
resource "castai_reservations" "test" {
	reservations_csv = <<EOF
Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
VM_RI_01-01-2023_01-01,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,1,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
VM_RI_01-01-2023_01-02,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:01Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,2,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview
	EOF
}
`
}

func makeAzureUpdatedReservationsConfig() string {
	return `
resource "castai_reservations" "test" {
	reservations_csv = <<EOF
Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
VM_RI_01-01-2023_01-01,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview
VM_RI_01-01-2023_01-02,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:01Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,2,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview
VM_RI_01-01-2023_01-03,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:02Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,1,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/1745741b-f3c6-46a9-ad16-b93775a1bc38/overview
	EOF
}
`
}

func makeGenericInitialReservationsConfig() string {
	return `
resource "castai_reservations" "test" {
	reservations_csv = <<EOF
name,provider,region,instance_type,price,count,start_date,end_date,zone_id,zone_name
reservation1,aws,us-east-1,c5n.large,,1,2020-01-01T00:00:00Z,2050-01-01T00:00:00Z,,
reservation2,aws,us-east-1,c5n.large,,2,2020-01-01T00:00:00Z,2050-01-01T00:00:01Z,,
	EOF
}
`
}

func makeGenericUpdatedReservationsConfig() string {
	return `
resource "castai_reservations" "test" {
	reservations_csv = <<EOF
name,provider,region,instance_type,price,count,start_date,end_date,zone_id,zone_name
reservation1,aws,us-east-1,c5n.large,,3,2020-01-01T00:00:00Z,2050-01-01T00:00:00Z,,
reservation2,aws,us-east-1,c5n.large,,2,2020-01-01T00:00:00Z,2050-01-01T00:00:01Z,,
reservation3,aws,us-east-1,c5n.large,,1,2020-01-01T00:00:00Z,2050-01-01T00:00:02Z,,
	EOF
}
`
}

func Test_getOrganizationId(t *testing.T) {
	organizationId1 := "7a704518-8275-4721-a622-18f4ec13fc22"
	organizationId2 := "be2ff71f-16ca-48ad-904d-9666baa37222"

	type args struct {
		organizationIdAttribute *string
		organizationsResponse   *sdk.OrganizationsList
	}
	tests := map[string]struct {
		args                     args
		want                     *string
		expectErrMessageContains *string
	}{
		"should use organization id property when it is provided": {
			args: args{
				organizationIdAttribute: lo.ToPtr(organizationId1),
			},
			want: lo.ToPtr(organizationId1),
		},
		"should use organization id from organizations list when organization is found": {
			args: args{
				organizationsResponse: &sdk.OrganizationsList{
					Organizations: []sdk.Organization{
						{
							Id: lo.ToPtr(organizationId2),
						},
					},
				},
			},
			want: lo.ToPtr(organizationId2),
		},
		"should return an error when organization id is not provided and more than one organization is found": {
			args: args{
				organizationsResponse: &sdk.OrganizationsList{
					Organizations: []sdk.Organization{
						{
							Id: lo.ToPtr(organizationId1),
						},
						{
							Id: lo.ToPtr(organizationId2),
						},
					},
				},
			},
			expectErrMessageContains: lo.ToPtr("found more than 1 organization, you can specify exact organization using 'organization_id' attribute"),
		},
	}

	for testName, tt := range tests {
		tt := tt

		t.Run(testName, func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)

			ctx := context.Background()
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			if tt.args.organizationsResponse != nil {
				organizationsResponseBytes, err := json.Marshal(tt.args.organizationsResponse)
				r.NoError(err)

				mockClient.EXPECT().
					ListOrganizations(gomock.Any()).
					Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(string(organizationsResponseBytes))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			}

			rReservations := resourceReservations()

			raw := make(map[string]interface{})
			if tt.args.organizationIdAttribute != nil {
				raw[reservations.FieldReservationsOrganizationId] = *tt.args.organizationIdAttribute
			}

			data := schema.TestResourceDataRaw(t, rReservations.Schema, raw)

			got, err := getOrganizationId(ctx, data, provider)

			if tt.expectErrMessageContains == nil {
				r.NoError(err)
				r.Equal(got, *tt.want)
			} else {
				r.Contains(err.Error(), *tt.expectErrMessageContains)
			}
		})
	}
}
