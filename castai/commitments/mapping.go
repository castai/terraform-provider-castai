package commitments

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type CommitmentsResource map[string]*string

func MapCsvRecordsToReservationResources(csvRecords [][]string) ([]*CommitmentsResource, error) {
	var csvColumns []string
	if len(csvRecords) > 0 {
		csvColumns = csvRecords[0]
	}
	normalizedCsvColumnNames := lo.Map(csvColumns, func(column string, _ int) string {
		return strings.ToLower(strings.ReplaceAll(column, " ", "_"))
	})

	reservationRecords := csvRecords[1:]
	fieldIndexes := mapReservationsHeaderToReservationFieldIndexes(normalizedCsvColumnNames)

	reservations := make([]*CommitmentsResource, 0, len(reservationRecords))
	for _, record := range reservationRecords {
		result, err := mapRecordToReservationResource(fieldIndexes, record)
		if err != nil {
			return nil, err
		}

		reservations = append(reservations, result)
	}
	return reservations, nil
}

func MapCommitmentToCommitmentsResource(c sdk.CastaiInventoryV1beta1Commitment) *CommitmentsResource {
	return &CommitmentsResource{
		FieldReservationName:      c.Name,
		FieldReservationRegion:    c.Region,
		FieldReservationStartDate: timeToString(c.StartDate),
		FieldReservationEndDate:   timeToString(c.EndDate),
	}
}

func MapCommitmentResourceToGCPCommitmentImport(resource CommitmentsResource) sdk.CastaiInventoryV1beta1GCPCommitmentImport {
	return sdk.CastaiInventoryV1beta1GCPCommitmentImport{}
}

func MapCommitmentResourceToAzureReservationImport(resource CommitmentsResource) sdk.CastaiInventoryV1beta1AzureReservationImport {
	return sdk.CastaiInventoryV1beta1AzureReservationImport{}
}

func MapToCommitmentResourcesWithCommonFieldsOnly(reservationResources []*CommitmentsResource) []*CommitmentsResource {
	return lo.Map(reservationResources, func(item *CommitmentsResource, _ int) *CommitmentsResource {
		return &CommitmentsResource{
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

func mapReservationsHeaderToReservationFieldIndexes(columns []string) map[string]int {
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

func mapRecordToReservationResource(fieldIndexes map[string]int, record []string) (*CommitmentsResource, error) {
	provider, err := getRecordReservationProvider(fieldIndexes, record)
	if err != nil {
		return nil, err
	}

	return &CommitmentsResource{
		FieldReservationName:                  getRecordFieldStringValue(FieldReservationName, fieldIndexes, record),
		FieldReservationProvider:              provider,
		FieldReservationRegion:                getRecordFieldStringValue(FieldReservationRegion, fieldIndexes, record),
		FieldReservationInstanceType:          getRecordFieldStringValue(FieldReservationInstanceType, fieldIndexes, record),
		FieldReservationPrice:                 getRecordFieldStringValue(FieldReservationPrice, fieldIndexes, record),
		FieldReservationCount:                 getRecordFieldStringValue(FieldReservationCount, fieldIndexes, record),
		FieldReservationStartDate:             getRecordFieldStringValue(FieldReservationStartDate, fieldIndexes, record),
		FieldReservationEndDate:               getRecordFieldStringValue(FieldReservationEndDate, fieldIndexes, record),
		FieldReservationZoneId:                getRecordFieldStringValue(FieldReservationZoneId, fieldIndexes, record),
		FieldReservationZoneName:              getRecordFieldStringValue(FieldReservationZoneName, fieldIndexes, record),
		FieldReservationProductName:           getRecordFieldStringValue(FieldReservationProductName, fieldIndexes, record),
		FieldReservationQuantity:              getRecordFieldStringValue(FieldReservationQuantity, fieldIndexes, record),
		FieldReservationPurchaseDate:          getRecordFieldStringValue(FieldReservationPurchaseDate, fieldIndexes, record),
		FieldReservationExpirationDate:        getRecordFieldStringValue(FieldReservationExpirationDate, fieldIndexes, record),
		FieldReservationType:                  getRecordFieldStringValue(FieldReservationType, fieldIndexes, record),
		FieldReservationDeepLinkToReservation: getRecordFieldStringValue(FieldReservationDeepLinkToReservation, fieldIndexes, record),
	}, nil
}

func getRecordReservationProvider(fieldIndexes map[string]int, record []string) (*string, error) {
	provider := getRecordFieldStringValue(FieldReservationProvider, fieldIndexes, record)
	if provider != nil && *provider != "" {
		return provider, nil
	}

	deepLinkToReservation := getRecordFieldStringValue(FieldReservationDeepLinkToReservation, fieldIndexes, record)
	if deepLinkToReservation != nil && strings.Contains(*deepLinkToReservation, "azure") {
		return lo.ToPtr("azure"), nil
	}

	return nil, fmt.Errorf("reservation provider could not be determined: %v", record)
}

func getRecordFieldStringValue(field string, fieldIndexes map[string]int, record []string) *string {
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
