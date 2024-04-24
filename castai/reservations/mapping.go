package reservations

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type ReservationResource map[string]*string

func MapCsvRecordsToReservationResources(csvRecords [][]string) ([]*ReservationResource, error) {
	var csvColumns []string
	if len(csvRecords) > 0 {
		csvColumns = csvRecords[0]
	}
	normalizedCsvColumnNames := lo.Map(csvColumns, func(column string, _ int) string {
		return strings.ToLower(strings.ReplaceAll(column, " ", "_"))
	})

	reservationRecords := csvRecords[1:]
	fieldIndexes := MapReservationsHeaderToReservationFieldIndexes(normalizedCsvColumnNames)

	reservations := make([]*ReservationResource, 0, len(reservationRecords))
	for _, record := range reservationRecords {
		result, err := mapRecordToReservationResource(fieldIndexes, record)
		if err != nil {
			return nil, err
		}

		reservations = append(reservations, result)
	}
	return reservations, nil
}

func MapReservationDetailsToReservationResource(reservation sdk.CastaiInventoryV1beta1ReservationDetails) *ReservationResource {
	return &ReservationResource{
		FieldReservationName:         reservation.Name,
		FieldReservationProvider:     reservation.Provider,
		FieldReservationRegion:       reservation.Region,
		FieldReservationInstanceType: reservation.InstanceType,
		FieldReservationPrice:        reservation.Price,
		FieldReservationCount:        int32ToString(reservation.Count),
		FieldReservationStartDate:    timeToString(reservation.StartDate),
		FieldReservationEndDate:      timeToString(reservation.EndDate),
		FieldReservationZoneId:       reservation.ZoneId,
		FieldReservationZoneName:     reservation.ZoneName,
	}
}

func MapReservationResourceToGenericReservation(resource ReservationResource) sdk.CastaiInventoryV1beta1GenericReservation {
	return sdk.CastaiInventoryV1beta1GenericReservation{
		Name:                  resource[FieldReservationName],
		Provider:              resource[FieldReservationProvider],
		Region:                resource[FieldReservationRegion],
		InstanceType:          resource[FieldReservationInstanceType],
		Price:                 resource[FieldReservationPrice],
		Count:                 stringToInt32(resource[FieldReservationCount]),
		StartDate:             stringToTime(resource[FieldReservationStartDate]),
		EndDate:               stringToTime(resource[FieldReservationEndDate]),
		ZoneId:                resource[FieldReservationZoneId],
		ZoneName:              resource[FieldReservationZoneName],
		Type:                  resource[FieldReservationType],
		DeepLinkToReservation: resource[FieldReservationDeepLinkToReservation],
		ProductName:           resource[FieldReservationProductName],
		ExpirationDate:        stringToTime(resource[FieldReservationExpirationDate]),
		PurchaseDate:          stringToTime(resource[FieldReservationPurchaseDate]),
		Quantity:              stringToInt32(resource[FieldReservationQuantity]),
	}
}

func MapToReservationResourcesWithCommonFieldsOnly(reservationResources []*ReservationResource) []*ReservationResource {
	return lo.Map(reservationResources, func(item *ReservationResource, _ int) *ReservationResource {
		return &ReservationResource{
			FieldReservationName:         (*item)[FieldReservationName],
			FieldReservationProvider:     (*item)[FieldReservationProvider],
			FieldReservationRegion:       (*item)[FieldReservationRegion],
			FieldReservationInstanceType: (*item)[FieldReservationInstanceType],
			FieldReservationPrice:        (*item)[FieldReservationPrice],
			FieldReservationCount:        (*item)[FieldReservationCount],
			FieldReservationStartDate:    (*item)[FieldReservationStartDate],
			FieldReservationEndDate:      (*item)[FieldReservationEndDate],
			FieldReservationZoneId:       (*item)[FieldReservationZoneId],
			FieldReservationZoneName:     (*item)[FieldReservationZoneName],
		}
	})
}

func MapReservationsHeaderToReservationFieldIndexes(columns []string) map[string]int {
	indexes := make(map[string]int, len(reservationResourceFields))
	for _, field := range reservationResourceFields {
		index := -1
		aliases := csvColumnAlias[field]
		for _, alias := range aliases {
			_, fieldIdx, found := lo.FindIndexOf(columns, func(item string) bool {
				return strings.ToLower(item) == alias
			})

			if found {
				index = fieldIdx
				break
			}
		}

		indexes[field] = index
	}
	return indexes
}

func mapRecordToReservationResource(fieldIndexes map[string]int, record []string) (*ReservationResource, error) {
	provider, err := GetRecordReservationProvider(fieldIndexes, record)
	if err != nil {
		return nil, err
	}

	return &ReservationResource{
		FieldReservationName:                  GetRecordFieldStringValue(FieldReservationName, fieldIndexes, record),
		FieldReservationProvider:              provider,
		FieldReservationRegion:                GetRecordFieldStringValue(FieldReservationRegion, fieldIndexes, record),
		FieldReservationInstanceType:          GetRecordFieldStringValue(FieldReservationInstanceType, fieldIndexes, record),
		FieldReservationPrice:                 GetRecordFieldStringValue(FieldReservationPrice, fieldIndexes, record),
		FieldReservationCount:                 GetRecordFieldStringValue(FieldReservationCount, fieldIndexes, record),
		FieldReservationStartDate:             GetRecordFieldStringValue(FieldReservationStartDate, fieldIndexes, record),
		FieldReservationEndDate:               GetRecordFieldStringValue(FieldReservationEndDate, fieldIndexes, record),
		FieldReservationZoneId:                GetRecordFieldStringValue(FieldReservationZoneId, fieldIndexes, record),
		FieldReservationZoneName:              GetRecordFieldStringValue(FieldReservationZoneName, fieldIndexes, record),
		FieldReservationProductName:           GetRecordFieldStringValue(FieldReservationProductName, fieldIndexes, record),
		FieldReservationQuantity:              GetRecordFieldStringValue(FieldReservationQuantity, fieldIndexes, record),
		FieldReservationPurchaseDate:          GetRecordFieldStringValue(FieldReservationPurchaseDate, fieldIndexes, record),
		FieldReservationExpirationDate:        GetRecordFieldStringValue(FieldReservationExpirationDate, fieldIndexes, record),
		FieldReservationType:                  GetRecordFieldStringValue(FieldReservationType, fieldIndexes, record),
		FieldReservationDeepLinkToReservation: GetRecordFieldStringValue(FieldReservationDeepLinkToReservation, fieldIndexes, record),
	}, nil
}

func GetRecordReservationProvider(fieldIndexes map[string]int, record []string) (*string, error) {
	provider := GetRecordFieldStringValue(FieldReservationProvider, fieldIndexes, record)
	if provider != nil && *provider != "" {
		return provider, nil
	}

	deepLinkToReservation := GetRecordFieldStringValue(FieldReservationDeepLinkToReservation, fieldIndexes, record)
	if deepLinkToReservation != nil && strings.Contains(*deepLinkToReservation, "azure") {
		return lo.ToPtr("azure"), nil
	}

	return nil, fmt.Errorf("reservation provider could not be determined: %v", record)
}

func GetRecordFieldStringValue(field string, fieldIndexes map[string]int, record []string) *string {
	index, found := fieldIndexes[field]
	if !found || index == -1 {
		return nil
	}
	value := record[index]
	if value == "" {
		return nil
	}

	return &value
}

func timeToString(t *time.Time) *string {
	if t == nil {
		return nil
	}

	result := t.Format(time.RFC3339)

	return &result
}

func stringToInt32(t *string) *int32 {
	if t == nil || *t == "" {
		return nil
	}

	parsed, err := strconv.Atoi(*t)
	if err != nil {
		return nil
	}
	result := int32(parsed)

	return &result
}

func int32ToString(t *int32) *string {
	if t == nil {
		return nil
	}

	result := strconv.Itoa(int(*t))

	return &result
}

func stringToTime(t *string) *time.Time {
	if t == nil || *t == "" {
		return nil
	}

	result, err := time.Parse(time.RFC3339, *t)
	if err != nil {
		return nil
	}

	return &result
}
