package commitments

import (
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/reservations"
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type (
	// Terraform SDK's diff setter uses mapstructure under the hood

	GCPCUDResource struct {
		Id             *string `mapstructure:"id"`
		Name           *string `mapstructure:"name"`
		Type           *string `mapstructure:"type"`
		Status         *string `mapstructure:"status"`
		StartTimestamp *string `mapstructure:"start_timestamp"`
		EndTimestamp   *string `mapstructure:"end_timestamp"`
		Plan           *string `mapstructure:"plan"`
		Region         *string `mapstructure:"region"`
		CPU            *int    `mapstructure:"cpu"`
		MemoryMb       *int    `mapstructure:"memory_mb"`
	}

	AzureReservationResource struct {
		Id             *string `mapstructure:"id"`
		Name           *string `mapstructure:"name"`
		Status         *string `mapstructure:"status"`
		StartTimestamp *string `mapstructure:"start_timestamp"`
		EndTimestamp   *string `mapstructure:"end_timestamp"`
		Region         *string `mapstructure:"region"`
		Count          *string `mapstructure:"count"`
		InstanceType   *string `mapstructure:"instance_type"`
		CPU            *string `mapstructure:"cpu"`
		MemoryMb       *string `mapstructure:"memory_mb"`
	}
)

func MapReservationCSVRecordsToResources(csvRecords [][]string) ([]*AzureReservationResource, error) {
	var csvColumns []string
	if len(csvRecords) > 0 {
		csvColumns = csvRecords[0]
	}
	normalizedCsvColumnNames := lo.Map(csvColumns, func(column string, _ int) string {
		return strings.ToLower(strings.ReplaceAll(column, " ", "_"))
	})

	reservationRecords := csvRecords[1:]
	fieldIndexes := mapReservationsCSVHeaderToFieldIndexes(normalizedCsvColumnNames)

	reservations := make([]*AzureReservationResource, 0, len(reservationRecords))
	for _, record := range reservationRecords {
		result, err := mapReservationRecordToResource(fieldIndexes, record)
		if err != nil {
			return nil, err
		}

		reservations = append(reservations, result)
	}
	return reservations, nil
}

func MapCommitmentToCUDResource(c sdk.CastaiInventoryV1beta1Commitment) (*GCPCUDResource, error) {
	var cpu, memory *int
	if c.GcpResourceCudContext != nil {
		if c.GcpResourceCudContext.Cpu != nil {
			parsedCPU, err := strconv.Atoi(*c.GcpResourceCudContext.Cpu)
			if err != nil {
				return nil, err
			}
			cpu = &parsedCPU
		}
		if c.GcpResourceCudContext.MemoryMb != nil {
			parsedMemory, err := strconv.Atoi(*c.GcpResourceCudContext.MemoryMb)
			if err != nil {
				return nil, err
			}
			memory = &parsedMemory
		}
	}

	return &GCPCUDResource{
		EndTimestamp:   timeToString(c.EndDate),
		Id:             c.GcpResourceCudContext.CudId,
		Name:           c.Name,
		Plan:           (*string)(c.GcpResourceCudContext.Plan),
		Region:         c.Region,
		CPU:            cpu,
		MemoryMb:       memory,
		StartTimestamp: timeToString(c.StartDate),
		Status:         (*string)(c.Status),
		Type:           c.GcpResourceCudContext.Type,
	}, nil
}

func MapCUDImportToResource(resource sdk.CastaiInventoryV1beta1GCPCommitmentImport) (*GCPCUDResource, error) {
	var cpu, memory *int
	if resource.Resources != nil {
		for _, res := range *resource.Resources {
			switch *res.Type {
			case "VCPU":
				parsedCPU, err := strconv.Atoi(*res.Amount)
				if err != nil {
					return nil, err
				}
				cpu = &parsedCPU
			case "MEMORY":
				parsedMemory, err := strconv.Atoi(*res.Amount)
				if err != nil {
					return nil, err
				}
				memory = &parsedMemory
			}
		}
	}

	return &GCPCUDResource{
		EndTimestamp:   resource.EndTimestamp,
		Id:             resource.Id,
		Name:           resource.Name,
		Plan:           resource.Plan,
		Region:         resource.Region,
		CPU:            cpu,
		MemoryMb:       memory,
		StartTimestamp: resource.StartTimestamp,
		Status:         resource.Status,
		Type:           resource.Type,
	}, nil
}

func mapReservationsCSVHeaderToFieldIndexes(columns []string) map[string]int {
	indexes := make(map[string]int, len(reservations.ReservationResourceFields))
	for _, field := range reservations.ReservationResourceFields {
		index := -1
		aliases := reservations.CSVColumnAlias[field]
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

func mapReservationRecordToResource(fieldIndexes map[string]int, record []string) (*AzureReservationResource, error) {
	return &AzureReservationResource{
		Id:             nil,
		Name:           nil,
		Status:         nil,
		StartTimestamp: nil,
		EndTimestamp:   nil,
		Region:         getRecordFieldStringValue(reservations.FieldReservationRegion, fieldIndexes, record),
		Count:          nil,
		InstanceType:   nil,
		CPU:            nil,
		MemoryMb:       nil,
	}, nil
}

func MapReservationsCSVRecordsToImports(csvRecords [][]string) ([]sdk.CastaiInventoryV1beta1AzureReservationImport, error) {
	var csvColumns []string
	if len(csvRecords) > 0 {
		csvColumns = csvRecords[0]
	}
	normalizedCsvColumnNames := lo.Map(csvColumns, func(column string, _ int) string {
		return strings.ToLower(strings.ReplaceAll(column, " ", "_"))
	})

	reservationRecords := csvRecords[1:]
	fieldIndexes := mapReservationsCSVHeaderToFieldIndexes(normalizedCsvColumnNames)

	res := make([]sdk.CastaiInventoryV1beta1AzureReservationImport, 0, len(reservationRecords))
	for _, record := range reservationRecords {
		result, err := MapReservationRecordToImport(fieldIndexes, record)
		if err != nil {
			return nil, err
		}

		res = append(res, *result)
	}
	return res, nil
}

func MapReservationRecordToImport(fieldIndexes map[string]int, record []string) (*sdk.CastaiInventoryV1beta1AzureReservationImport, error) {
	var quantity *int32
	if v := getRecordFieldStringValue(reservations.FieldReservationQuantity, fieldIndexes, record); v != nil {
		parsed, err := strconv.Atoi(*v)
		if err != nil {
			return nil, err
		}
		quantity = lo.ToPtr(int32(parsed))
	}

	return &sdk.CastaiInventoryV1beta1AzureReservationImport{
		ExpirationDate: getRecordFieldStringValue(reservations.FieldReservationExpirationDate, fieldIndexes, record),
		Name:           getRecordFieldStringValue(reservations.FieldReservationName, fieldIndexes, record),
		ProductName:    getRecordFieldStringValue(reservations.FieldReservationProductName, fieldIndexes, record),
		PurchaseDate:   getRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		Quantity:       quantity,
		Region:         getRecordFieldStringValue(reservations.FieldReservationRegion, fieldIndexes, record),
		//ReservationId:      getRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		//Scope:              getRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		ScopeResourceGroup: nil,
		ScopeSubscription:  nil,
		//Status:             getRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		//Term:               getRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		Type: getRecordFieldStringValue(reservations.FieldReservationType, fieldIndexes, record),
	}, nil
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
	return lo.ToPtr(t.Format(time.RFC3339))
}
