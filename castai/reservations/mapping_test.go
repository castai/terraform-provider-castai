package reservations

import (
	"testing"
	"time"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestMapCsvRecordsToReservationResources(t *testing.T) {
	type args struct {
		csvRecords [][]string
	}
	tests := map[string]struct {
		args                     args
		want                     []*ReservationResource
		expectErrMessageContains *string
	}{
		"should map reservation resources when generic reservation records are provided": {
			args: args{
				csvRecords: [][]string{
					{"name", "provider", "region", "instance_type", "price", "count", "start_date", "end_date", "zone_id", "zone_name"},
					{"reservation1", "aws", "us-east-1", "c5n.large", "", "3", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
					{"reservation2", "aws", "us-east-1", "c5n.large", "", "2", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
					{"reservation3", "aws", "us-east-1", "c5n.large", "", "1", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
				},
			},
			want: []*ReservationResource{
				{
					FieldReservationName:                  lo.ToPtr("reservation1"),
					FieldReservationProvider:              lo.ToPtr("aws"),
					FieldReservationRegion:                lo.ToPtr("us-east-1"),
					FieldReservationInstanceType:          lo.ToPtr("c5n.large"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("3"),
					FieldReservationStartDate:             lo.ToPtr("2020-01-01T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           nil,
					FieldReservationQuantity:              nil,
					FieldReservationPurchaseDate:          nil,
					FieldReservationExpirationDate:        nil,
					FieldReservationType:                  nil,
					FieldReservationDeepLinkToReservation: nil,
				},
				{
					FieldReservationName:                  lo.ToPtr("reservation2"),
					FieldReservationProvider:              lo.ToPtr("aws"),
					FieldReservationRegion:                lo.ToPtr("us-east-1"),
					FieldReservationInstanceType:          lo.ToPtr("c5n.large"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("2"),
					FieldReservationStartDate:             lo.ToPtr("2020-01-01T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           nil,
					FieldReservationQuantity:              nil,
					FieldReservationPurchaseDate:          nil,
					FieldReservationExpirationDate:        nil,
					FieldReservationType:                  nil,
					FieldReservationDeepLinkToReservation: nil,
				},
				{
					FieldReservationName:                  lo.ToPtr("reservation3"),
					FieldReservationProvider:              lo.ToPtr("aws"),
					FieldReservationRegion:                lo.ToPtr("us-east-1"),
					FieldReservationInstanceType:          lo.ToPtr("c5n.large"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("1"),
					FieldReservationStartDate:             lo.ToPtr("2020-01-01T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           nil,
					FieldReservationQuantity:              nil,
					FieldReservationPurchaseDate:          nil,
					FieldReservationExpirationDate:        nil,
					FieldReservationType:                  nil,
					FieldReservationDeepLinkToReservation: nil,
				},
			},
		},
		"should map reservation resources when Azure reservation records are provided": {
			args: args{
				csvRecords: [][]string{
					{"Name", "Reservation Id", "Reservation order Id", "Status", "Expiration date", "Purchase date", "Term", "Scope", "Scope subscription", "Scope resource group", "Type", "Product name", "Region", "Quantity", "Utilization % 1 Day", "Utilization % 7 Day", "Utilization % 30 Day", "Deep link to reservation"},
					{"VM_RI_01-01-2023_01-01", "3b3de39c-bc44-4d69-be2d-69527dfe9958", "630226bb-5170-4b95-90b0-f222757130c1", "Succeeded", "2050-01-01T00:00:00Z", "2023-01-11T00:00:00Z", "P3Y", "Single subscription", "8faa0959-093b-4612-8686-a996ac19db00", "All resource groups", "VirtualMachines", "Standard_D32as_v4", "eastus", "3", "100", "100", "100", "https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview"},
					{"VM_RI_01-01-2023_01-02", "3b3de39c-bc44-4d69-be2d-69527dfe9958", "630226bb-5170-4b95-90b0-f222757130c1", "Succeeded", "2050-01-01T00:00:00Z", "2023-01-11T00:00:00Z", "P3Y", "Single subscription", "8faa0959-093b-4612-8686-a996ac19db00", "All resource groups", "VirtualMachines", "Standard_D32as_v4", "eastus", "2", "100", "100", "100", "https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview"},
					{"VM_RI_01-01-2023_01-03", "3b3de39c-bc44-4d69-be2d-69527dfe9958", "630226bb-5170-4b95-90b0-f222757130c1", "Succeeded", "2050-01-01T00:00:00Z", "2023-01-11T00:00:00Z", "P3Y", "Single subscription", "8faa0959-093b-4612-8686-a996ac19db00", "All resource groups", "VirtualMachines", "Standard_D32as_v4", "eastus", "1", "100", "100", "100", "https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/1745741b-f3c6-46a9-ad16-b93775a1bc38/overview"},
				},
			},
			want: []*ReservationResource{
				{
					FieldReservationName:                  lo.ToPtr("VM_RI_01-01-2023_01-01"),
					FieldReservationProvider:              lo.ToPtr("azure"),
					FieldReservationRegion:                lo.ToPtr("eastus"),
					FieldReservationInstanceType:          lo.ToPtr("Standard_D32as_v4"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("3"),
					FieldReservationStartDate:             lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           lo.ToPtr("Standard_D32as_v4"),
					FieldReservationQuantity:              lo.ToPtr("3"),
					FieldReservationPurchaseDate:          lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationExpirationDate:        lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationType:                  lo.ToPtr("VirtualMachines"),
					FieldReservationDeepLinkToReservation: lo.ToPtr("https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview"),
				},
				{
					FieldReservationName:                  lo.ToPtr("VM_RI_01-01-2023_01-02"),
					FieldReservationProvider:              lo.ToPtr("azure"),
					FieldReservationRegion:                lo.ToPtr("eastus"),
					FieldReservationInstanceType:          lo.ToPtr("Standard_D32as_v4"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("2"),
					FieldReservationStartDate:             lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           lo.ToPtr("Standard_D32as_v4"),
					FieldReservationQuantity:              lo.ToPtr("2"),
					FieldReservationPurchaseDate:          lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationExpirationDate:        lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationType:                  lo.ToPtr("VirtualMachines"),
					FieldReservationDeepLinkToReservation: lo.ToPtr("https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/25b95bdb-b78b-4973-a60c-71e70f158eca/overview"),
				},
				{
					FieldReservationName:                  lo.ToPtr("VM_RI_01-01-2023_01-03"),
					FieldReservationProvider:              lo.ToPtr("azure"),
					FieldReservationRegion:                lo.ToPtr("eastus"),
					FieldReservationInstanceType:          lo.ToPtr("Standard_D32as_v4"),
					FieldReservationPrice:                 nil,
					FieldReservationCount:                 lo.ToPtr("1"),
					FieldReservationStartDate:             lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationEndDate:               lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationZoneId:                nil,
					FieldReservationZoneName:              nil,
					FieldReservationProductName:           lo.ToPtr("Standard_D32as_v4"),
					FieldReservationQuantity:              lo.ToPtr("1"),
					FieldReservationPurchaseDate:          lo.ToPtr("2023-01-11T00:00:00Z"),
					FieldReservationExpirationDate:        lo.ToPtr("2050-01-01T00:00:00Z"),
					FieldReservationType:                  lo.ToPtr("VirtualMachines"),
					FieldReservationDeepLinkToReservation: lo.ToPtr("https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/1745741b-f3c6-46a9-ad16-b93775a1bc38/overview"),
				},
			},
		},
		"should return an error when reservation provider could not be determined": {
			args: args{
				csvRecords: [][]string{
					{"name", "provider", "region", "instance_type", "price", "count", "start_date", "end_date", "zone_id", "zone_name"},
					{"reservation1", "aws", "us-east-1", "c5n.large", "", "3", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
					{"reservation2", "", "us-east-1", "c5n.large", "", "2", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
					{"reservation3", "aws", "us-east-1", "c5n.large", "", "1", "2020-01-01T00:00:00Z", "2050-01-01T00:00:00Z", "", ""},
				},
			},
			expectErrMessageContains: lo.ToPtr("reservation provider could not be determined: [reservation2"),
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			r := require.New(t)

			got, err := MapCsvRecordsToReservationResources(tt.args.csvRecords)

			if tt.expectErrMessageContains != nil {
				r.Error(err)
				r.Contains(err.Error(), *tt.expectErrMessageContains)
			} else {
				r.NoError(err)
				r.ElementsMatch(tt.want, got)
			}
		})
	}
}

func TestMapReservationDetailsToReservationResource(t *testing.T) {
	now := time.Now()
	commonFields := []string{
		FieldReservationName,
		FieldReservationProvider,
		FieldReservationRegion,
		FieldReservationInstanceType,
		FieldReservationPrice,
		FieldReservationCount,
		FieldReservationStartDate,
		FieldReservationEndDate,
		FieldReservationZoneId,
		FieldReservationZoneName,
	}

	type args struct {
		reservation sdk.CastaiInventoryV1beta1ReservationDetails
	}
	tests := map[string]struct {
		args args
		want *ReservationResource
	}{
		"should map common fields from reservation": {
			args: args{
				reservation: sdk.CastaiInventoryV1beta1ReservationDetails{
					Name:         lo.ToPtr("reservation"),
					Provider:     lo.ToPtr("gcp"),
					Region:       lo.ToPtr("region"),
					InstanceType: lo.ToPtr("it"),
					Price:        lo.ToPtr("1"),
					Count:        lo.ToPtr(int32(1)),
					StartDate:    lo.ToPtr(now),
					EndDate:      lo.ToPtr(now),
					ZoneId:       lo.ToPtr("zone id"),
					ZoneName:     lo.ToPtr("zone name"),
				},
			},
			want: &ReservationResource{
				FieldReservationName:         lo.ToPtr("reservation"),
				FieldReservationProvider:     lo.ToPtr("gcp"),
				FieldReservationRegion:       lo.ToPtr("region"),
				FieldReservationInstanceType: lo.ToPtr("it"),
				FieldReservationPrice:        lo.ToPtr("1"),
				FieldReservationCount:        lo.ToPtr("1"),
				FieldReservationStartDate:    lo.ToPtr(now.Format(time.RFC3339)),
				FieldReservationEndDate:      lo.ToPtr(now.Format(time.RFC3339)),
				FieldReservationZoneId:       lo.ToPtr("zone id"),
				FieldReservationZoneName:     lo.ToPtr("zone name"),
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			r := require.New(t)
			got := MapReservationDetailsToReservationResource(tt.args.reservation)

			r.Equal(tt.want, got)
			for key, value := range *got {
				if !lo.Contains(commonFields, key) && value != nil {
					r.Nilf(value, "expected '%s' to not be set", key)
				}
			}
		})
	}
}

func Test_mapReservationsHeaderToReservationFieldIndexes(t *testing.T) {
	type args struct {
		columns []string
	}
	tests := map[string]struct {
		args args
		want map[string]int
	}{
		"should map field indexes when column names match primary field alias": {
			args: args{
				columns: []string{
					FieldReservationName,
					FieldReservationProvider,
					FieldReservationRegion,
					FieldReservationInstanceType,
					FieldReservationPrice,
					FieldReservationCount,
					FieldReservationStartDate,
					FieldReservationEndDate,
					FieldReservationZoneId,
					FieldReservationZoneName,
					FieldReservationProductName,
					FieldReservationQuantity,
					FieldReservationPurchaseDate,
					FieldReservationExpirationDate,
					FieldReservationType,
					FieldReservationDeepLinkToReservation,
					FieldReservationID,
					FieldReservationScopeResourceGroup,
					FieldReservationScopeSubscription,
					FieldReservationScopeStatus,
					FieldReservationTerm,
					FieldReservationStatus,
					FieldReservationScope,
				},
			},
			want: map[string]int{
				FieldReservationName:                  0,
				FieldReservationProvider:              1,
				FieldReservationRegion:                2,
				FieldReservationInstanceType:          3,
				FieldReservationPrice:                 4,
				FieldReservationCount:                 5,
				FieldReservationStartDate:             6,
				FieldReservationEndDate:               7,
				FieldReservationZoneId:                8,
				FieldReservationZoneName:              9,
				FieldReservationProductName:           10,
				FieldReservationQuantity:              11,
				FieldReservationPurchaseDate:          12,
				FieldReservationExpirationDate:        13,
				FieldReservationType:                  14,
				FieldReservationDeepLinkToReservation: 15,
				FieldReservationID:                    16,
				FieldReservationScopeResourceGroup:    17,
				FieldReservationScopeSubscription:     18,
				FieldReservationScopeStatus:           19,
				FieldReservationTerm:                  20,
				FieldReservationStatus:                21,
				FieldReservationScope:                 22,
			},
		},
		"should map field indexes when column names match secondary alias field": {
			args: args{
				columns: []string{
					FieldReservationName,
					FieldReservationProvider,
					FieldReservationRegion,
					FieldReservationPrice,
					FieldReservationZoneId,
					FieldReservationZoneName,
					FieldReservationProductName,
					FieldReservationQuantity,
					FieldReservationPurchaseDate,
					FieldReservationExpirationDate,
					FieldReservationType,
					FieldReservationDeepLinkToReservation,
					FieldReservationID,
					FieldReservationScopeResourceGroup,
					FieldReservationScopeSubscription,
					FieldReservationScopeStatus,
					FieldReservationTerm,
					FieldReservationStatus,
					FieldReservationScope,
				},
			},
			want: map[string]int{
				FieldReservationName:                  0,
				FieldReservationProvider:              1,
				FieldReservationRegion:                2,
				FieldReservationPrice:                 3,
				FieldReservationZoneId:                4,
				FieldReservationZoneName:              5,
				FieldReservationInstanceType:          6,
				FieldReservationProductName:           6,
				FieldReservationQuantity:              7,
				FieldReservationCount:                 7,
				FieldReservationStartDate:             8,
				FieldReservationPurchaseDate:          8,
				FieldReservationEndDate:               9,
				FieldReservationExpirationDate:        9,
				FieldReservationType:                  10,
				FieldReservationDeepLinkToReservation: 11,
				FieldReservationID:                    12,
				FieldReservationScopeResourceGroup:    13,
				FieldReservationScopeSubscription:     14,
				FieldReservationScopeStatus:           15,
				FieldReservationTerm:                  16,
				FieldReservationStatus:                17,
				FieldReservationScope:                 18,
			},
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			r := require.New(t)

			got := MapReservationsHeaderToReservationFieldIndexes(tt.args.columns)

			r.Equal(tt.want, got)
		})
	}
}

func Test_getRecordReservationProvider(t *testing.T) {
	type args struct {
		fieldIndexes map[string]int
		record       []string
	}
	tests := map[string]struct {
		args                     args
		want                     *string
		expectErrMessageContains *string
	}{
		"should use aws provider when record provider is aws": {
			args: args{
				fieldIndexes: map[string]int{
					FieldReservationProvider: 0,
				},
				record: []string{"aws"},
			},
			want: lo.ToPtr("aws"),
		},
		"should use azure provider when record provider is azure": {
			args: args{
				fieldIndexes: map[string]int{
					FieldReservationProvider: 0,
				},
				record: []string{"azure"},
			},
			want: lo.ToPtr("azure"),
		},
		"should use gcp provider when record provider is gcp": {
			args: args{
				fieldIndexes: map[string]int{
					FieldReservationProvider: 0,
				},
				record: []string{"gcp"},
			},
			want: lo.ToPtr("gcp"),
		},
		"should determine azure provider when deep link to reservation is provided": {
			args: args{
				fieldIndexes: map[string]int{
					FieldReservationDeepLinkToReservation: 0,
				},
				record: []string{"https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview"},
			},
			want: lo.ToPtr("azure"),
		},
		"should return an error if provider could not be determined": {
			args: args{
				fieldIndexes: map[string]int{},
				record:       []string{},
			},
			expectErrMessageContains: lo.ToPtr("reservation provider could not be determined"),
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			r := require.New(t)

			got, err := GetRecordReservationProvider(tt.args.fieldIndexes, tt.args.record)

			if tt.expectErrMessageContains != nil {
				r.Error(err)
				r.Contains(err.Error(), *tt.expectErrMessageContains)
			} else {
				r.Equal(tt.want, got)
			}
		})
	}
}
