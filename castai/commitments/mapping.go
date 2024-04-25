package commitments

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/castai/terraform-provider-castai/castai/reservations"
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type (
	// Terraform SDK's diff setter uses mapstructure under the hood

	GCPCUDResource struct {
		// CAST AI only fields
		ID             *string  `mapstructure:"id,omitempty"` // ID of the commitment
		AllowedUsage   *float32 `mapstructure:"allowed_usage,omitempty"`
		Prioritization *bool    `mapstructure:"prioritization,omitempty"`
		Status         *string  `mapstructure:"status,omitempty"`

		// Fields from GCP CUDs export JSON
		CUDID          string `mapstructure:"cud_id"` // ID of the CUD in GCP
		CUDStatus      string `mapstructure:"cud_status"`
		StartTimestamp string `mapstructure:"start_timestamp"`
		EndTimestamp   string `mapstructure:"end_timestamp"`
		Name           string `mapstructure:"name"`
		Region         string `mapstructure:"region"`
		CPU            int    `mapstructure:"cpu"`
		MemoryMb       int    `mapstructure:"memory_mb"`
		Plan           string `mapstructure:"plan"`
		Type           string `mapstructure:"type"`
	}

	AzureReservationResource struct {
		// CAST AI only fields
		ID             *string  `mapstructure:"id,omitempty"` // ID of the commitment
		AllowedUsage   *float32 `mapstructure:"allowed_usage,omitempty"`
		Prioritization *bool    `mapstructure:"prioritization,omitempty"`
		Status         *string  `mapstructure:"status,omitempty"`

		// Fields from Azure reservations export CSV
		Count              int    `mapstructure:"count"`
		ReservationID      string `mapstructure:"reservation_id"` // ID of the reservation in Azure
		ReservationStatus  string `mapstructure:"reservation_status"`
		StartTimestamp     string `mapstructure:"start_timestamp"`
		EndTimestamp       string `mapstructure:"end_timestamp"`
		Name               string `mapstructure:"name"`
		Region             string `mapstructure:"region"`
		InstanceType       string `mapstructure:"instance_type"`
		Plan               string `mapstructure:"plan"`
		Scope              string `mapstructure:"scope"`
		ScopeResourceGroup string `mapstructure:"scope_resource_group"`
		ScopeSubscription  string `mapstructure:"scope_subscription"`
	}

	// Resource is an interface for common management of GCP (GCPCUDResource) and Azure (AzureReservationResource) resources
	Resource interface {
		// GetCommitmentID returns the ID of the commitment in CAST AI
		GetCommitmentID() string
		// GetIDInCloud returns the ID of the resource in the cloud provider
		GetIDInCloud() string
	}

	CommitmentConfigResource struct {
		Matcher        CommitmentConfigMatcherResource `mapstructure:"matcher"`
		Prioritization *bool                           `mapstructure:"prioritization,omitempty"`
		Status         *string                         `mapstructure:"status,omitempty"`
		AllowedUsage   *float32                        `mapstructure:"allowed_usage,omitempty"`
	}
	CommitmentConfigMatcherResource struct {
		Name        string  `mapstructure:"name"`
		Type        *string `mapstructure:"type,omitempty"`
		ProductName *string `mapstructure:"product_name,omitempty"` // basically Type but for Azure
		Region      *string `mapstructure:"region,omitempty"`
	}
)

var (
	_ Resource = (*GCPCUDResource)(nil)
	_ Resource = (*AzureReservationResource)(nil)
)

func (r *GCPCUDResource) GetCommitmentID() string {
	if r == nil || r.ID == nil {
		return ""
	}
	return *r.ID
}

func (r *GCPCUDResource) GetIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.CUDID
}

func (r *AzureReservationResource) GetCommitmentID() string {
	if r == nil || r.ID == nil {
		return ""
	}
	return *r.ID
}

func (r *AzureReservationResource) GetIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.ReservationID
}

func (m CommitmentConfigMatcherResource) Validate() error {
	if m.Name == "" {
		return errors.New("matcher name is required")
	}
	return nil
}

func MapCommitmentToCUDResource(c sdk.CastaiInventoryV1beta1Commitment) (*GCPCUDResource, error) {
	if c.GcpResourceCudContext == nil {
		return nil, errors.New("missing GCP resource CUD context")
	}

	var cpu, memory int
	if c.GcpResourceCudContext.Cpu != nil {
		parsedCPU, err := strconv.Atoi(*c.GcpResourceCudContext.Cpu)
		if err != nil {
			return nil, err
		}
		cpu = parsedCPU
	}
	if c.GcpResourceCudContext.MemoryMb != nil {
		parsedMemory, err := strconv.Atoi(*c.GcpResourceCudContext.MemoryMb)
		if err != nil {
			return nil, err
		}
		memory = parsedMemory
	}

	var endDate, startDate string
	if c.EndDate != nil {
		endDate = c.EndDate.Format(time.RFC3339)
	}
	if c.StartDate != nil {
		startDate = c.StartDate.Format(time.RFC3339)
	}

	return &GCPCUDResource{
		ID:             c.Id,
		AllowedUsage:   c.AllowedUsage,
		Prioritization: c.Prioritization,
		Status:         (*string)(c.Status),
		CUDID:          lo.FromPtr(c.GcpResourceCudContext.CudId),
		CUDStatus:      lo.FromPtr(c.GcpResourceCudContext.Status),
		EndTimestamp:   endDate,
		StartTimestamp: startDate,
		Name:           lo.FromPtr(c.Name),
		Region:         lo.FromPtr(c.Region),
		CPU:            cpu,
		MemoryMb:       memory,
		Plan:           lo.FromPtr((*string)(c.GcpResourceCudContext.Plan)),
		Type:           lo.FromPtr(c.GcpResourceCudContext.Type),
	}, nil
}

func MapCommitmentToReservationResource(c sdk.CastaiInventoryV1beta1Commitment) (*AzureReservationResource, error) {
	if c.AzureReservationContext == nil {
		return nil, errors.New("missing azure resource reservation context")
	}

	var startDate, endDate string
	if c.StartDate != nil {
		startDate = c.StartDate.Format(time.RFC3339)
	}
	if c.EndDate != nil {
		endDate = c.EndDate.Format(time.RFC3339)
	}
	return &AzureReservationResource{
		ID:                 c.Id,
		AllowedUsage:       c.AllowedUsage,
		Prioritization:     c.Prioritization,
		Status:             (*string)(c.Status),
		Count:              int(lo.FromPtr(c.AzureReservationContext.Count)),
		ReservationID:      lo.FromPtr(c.AzureReservationContext.Id),
		ReservationStatus:  lo.FromPtr(c.AzureReservationContext.Status),
		StartTimestamp:     startDate,
		EndTimestamp:       endDate,
		Name:               lo.FromPtr(c.Name),
		Region:             lo.FromPtr(c.Region),
		InstanceType:       lo.FromPtr(c.AzureReservationContext.InstanceType),
		Plan:               string(lo.FromPtr(c.AzureReservationContext.Plan)),
		Scope:              lo.FromPtr(c.AzureReservationContext.Scope),
		ScopeResourceGroup: lo.FromPtr(c.AzureReservationContext.ScopeResourceGroup),
		ScopeSubscription:  lo.FromPtr(c.AzureReservationContext.ScopeSubscription),
	}, nil
}

func MapCUDImportToResource(
	cudWithCfg *commitmentWithConfig[CastaiGCPCommitmentImport],
) (*GCPCUDResource, error) {
	var cpu, memory int
	if cudWithCfg.Commitment.Resources != nil {
		for _, res := range *cudWithCfg.Commitment.Resources {
			if res.Type == nil || res.Amount == nil {
				continue
			}
			switch *res.Type {
			case "VCPU":
				parsedCPU, err := strconv.Atoi(*res.Amount)
				if err != nil {
					return nil, err
				}
				cpu = parsedCPU
			case "MEMORY":
				parsedMemory, err := strconv.Atoi(*res.Amount)
				if err != nil {
					return nil, err
				}
				memory = parsedMemory
			}
		}
	}

	// GCP's exports contain the full path of the region, we only need the region name. CAST AI's API does the same
	// thing so we need to do it here too in order to avoid Terraform diff mismatches.
	// Example region value: https://www.googleapis.com/compute/v1/projects/{PROJECT}/regions/{REGION}
	var region string
	if cudWithCfg.Commitment.Region != nil {
		_, region = path.Split(*cudWithCfg.Commitment.Region)
	}

	res := &GCPCUDResource{
		CUDID:          lo.FromPtr(cudWithCfg.Commitment.Id),
		CUDStatus:      lo.FromPtr(cudWithCfg.Commitment.Status),
		EndTimestamp:   lo.FromPtr(cudWithCfg.Commitment.EndTimestamp),
		StartTimestamp: lo.FromPtr(cudWithCfg.Commitment.StartTimestamp),
		Name:           lo.FromPtr(cudWithCfg.Commitment.Name),
		Region:         region,
		CPU:            cpu,
		MemoryMb:       memory,
		Plan:           lo.FromPtr(cudWithCfg.Commitment.Plan),
		Type:           lo.FromPtr(cudWithCfg.Commitment.Type),
	}
	if cudWithCfg.Config != nil {
		res.AllowedUsage = cudWithCfg.Config.AllowedUsage
		res.Prioritization = cudWithCfg.Config.Prioritization
		res.Status = cudWithCfg.Config.Status
	}
	return res, nil
}

func MapReservationImportToResource(
	cudWithCfg *commitmentWithConfig[CastaiAzureReservationImport],
) *AzureReservationResource {
	var plan string
	if cudWithCfg.Commitment.Term != nil {
		switch *cudWithCfg.Commitment.Term { // normalize the values just like CAST AI's API does
		case "P1Y":
			plan = "ONE_YEAR"
		case "P3Y":
			plan = "THREE_YEAR"
		case "ONE_YEAR":
		case "THREE_YEAR":
		default:
			plan = "ONE_YEAR"
		}
	}

	return &AzureReservationResource{
		Count:              int(lo.FromPtr(cudWithCfg.Commitment.Quantity)),
		ReservationID:      lo.FromPtr(cudWithCfg.Commitment.ReservationId),
		ReservationStatus:  lo.FromPtr(cudWithCfg.Commitment.Status),
		StartTimestamp:     lo.FromPtr(cudWithCfg.Commitment.PurchaseDate),
		EndTimestamp:       lo.FromPtr(cudWithCfg.Commitment.ExpirationDate),
		Name:               lo.FromPtr(cudWithCfg.Commitment.Name),
		Region:             lo.FromPtr(cudWithCfg.Commitment.Region),
		InstanceType:       lo.FromPtr(cudWithCfg.Commitment.ProductName),
		Plan:               plan,
		Scope:              lo.FromPtr(cudWithCfg.Commitment.Scope),
		ScopeResourceGroup: lo.FromPtr(cudWithCfg.Commitment.ScopeResourceGroup),
		ScopeSubscription:  lo.FromPtr(cudWithCfg.Commitment.ScopeSubscription),
	}
}

// commitmentConfigMatcherKey is a utility type for mapping CUDs to their configurations
type commitmentConfigMatcherKey struct {
	name, region, typ string
}

func (k commitmentConfigMatcherKey) String() string {
	return fmt.Sprintf("%s-%s-%s", k.name, k.region, k.typ)
}

// commitment is a common interface for CastaiGCPCommitmentImport and sdk.CastaiInventoryV1beta1Commitment
type commitment interface {
	getKey() commitmentConfigMatcherKey
}

type commitmentWithConfig[C commitment] struct {
	Commitment C
	Config     *CommitmentConfigResource
}

func MapConfigsToCommitments[C commitment](
	cmts []C,
	configs []*CommitmentConfigResource,
) ([]*commitmentWithConfig[C], error) {
	res := make([]*commitmentWithConfig[C], 0, len(cmts))
	configsByKey := map[commitmentConfigMatcherKey]*CommitmentConfigResource{}
	for _, c := range configs {
		var region string
		if c.Matcher.Region != nil {
			_, region = path.Split(*c.Matcher.Region)
		}
		key := commitmentConfigMatcherKey{
			name:   c.Matcher.Name,
			region: region,
			typ:    lo.FromPtr(c.Matcher.Type),
		}
		if _, ok := configsByKey[key]; ok { // Make sure each config matcher is unique
			return nil, fmt.Errorf("duplicate configuration for %s", key.String())
		}
		configsByKey[key] = c
	}

	var mappedConfigs int
	processedKeys := map[commitmentConfigMatcherKey]struct{}{}
	for _, cmt := range cmts {
		key := cmt.getKey()
		if _, ok := processedKeys[key]; ok { // Make sure each CUD is unique
			return nil, fmt.Errorf("duplicate import for %s", key.String())
		}
		processedKeys[key] = struct{}{}

		config, hasConfig := configsByKey[key]
		if hasConfig {
			mappedConfigs++
		}
		res = append(res, &commitmentWithConfig[C]{
			Commitment: cmt,
			Config:     config,
		})
	}
	if mappedConfigs != len(configs) { // Make sure all configs were mapped
		return nil, fmt.Errorf("not all commitment configurations were mapped")
	}
	return res, nil
}

func MapConfiguredCUDImportsToResources[C interface {
	CastaiGCPCommitmentImport | sdk.CastaiInventoryV1beta1GCPCommitmentImport
}](
	cuds []C,
	configs []*CommitmentConfigResource,
) ([]*GCPCUDResource, error) {
	if len(configs) > len(cuds) {
		return nil, fmt.Errorf("more configurations than CUDs")
	}

	var cudImports []CastaiGCPCommitmentImport
	switch v := any(cuds).(type) {
	case []CastaiGCPCommitmentImport:
		cudImports = v
	case []sdk.CastaiInventoryV1beta1GCPCommitmentImport:
		cudImports = make([]CastaiGCPCommitmentImport, 0, len(v))
		for _, item := range v {
			cudImports = append(cudImports, CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: item})
		}
	}

	cudsWithConfigs, err := MapConfigsToCommitments(cudImports, configs)
	if err != nil {
		return nil, err
	}

	res := make([]*GCPCUDResource, 0, len(cudsWithConfigs))
	for _, item := range cudsWithConfigs {
		v, err := MapCUDImportToResource(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

func MapConfiguredReservationImportsToResources[C interface {
	CastaiAzureReservationImport | sdk.CastaiInventoryV1beta1AzureReservationImport
}](
	reservations []C,
	configs []*CommitmentConfigResource,
) ([]*AzureReservationResource, error) {
	if len(configs) > len(reservations) {
		return nil, fmt.Errorf("more configurations than reservations")
	}

	var cudImports []CastaiAzureReservationImport
	switch v := any(reservations).(type) {
	case []CastaiAzureReservationImport:
		cudImports = v
	case []sdk.CastaiInventoryV1beta1AzureReservationImport:
		cudImports = make([]CastaiAzureReservationImport, 0, len(v))
		for _, item := range v {
			cudImports = append(cudImports, CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: item})
		}
	}

	cudsWithConfigs, err := MapConfigsToCommitments(cudImports, configs)
	if err != nil {
		return nil, err
	}

	return lo.Map(cudsWithConfigs, func(c *commitmentWithConfig[CastaiAzureReservationImport], _ int) *AzureReservationResource {
		return MapReservationImportToResource(c)
	}), nil
}

func MapCommitmentImportWithConfigToUpdateRequest(
	c *commitmentWithConfig[CastaiCommitment],
) sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody {
	req := sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
		AllowedUsage:            c.Commitment.AllowedUsage,
		EndDate:                 c.Commitment.EndDate,
		GcpResourceCudContext:   c.Commitment.GcpResourceCudContext,
		AzureReservationContext: c.Commitment.AzureReservationContext,
		Id:                      c.Commitment.Id,
		Name:                    c.Commitment.Name,
		Prioritization:          c.Commitment.Prioritization,
		Region:                  c.Commitment.Region,
		StartDate:               c.Commitment.StartDate,
		Status:                  c.Commitment.Status,
	}
	if c.Config != nil {
		if c.Config.AllowedUsage != nil {
			req.AllowedUsage = c.Config.AllowedUsage
		}
		if c.Config.Prioritization != nil {
			req.Prioritization = c.Config.Prioritization
		}
		if c.Config.Status != nil {
			req.Status = (*sdk.CastaiInventoryV1beta1CommitmentStatus)(c.Config.Status)
		}
	}
	return req
}

// Azure specific stuff

func MapReservationCSVRowsToImports(csvRecords [][]string) ([]sdk.CastaiInventoryV1beta1AzureReservationImport, error) {
	var csvColumns []string
	if len(csvRecords) > 0 {
		csvColumns = csvRecords[0]
	}
	normalizedCsvColumnNames := lo.Map(csvColumns, func(column string, _ int) string {
		return strings.ToLower(strings.ReplaceAll(column, " ", "_"))
	})

	reservationRecords := csvRecords[1:]
	fieldIndexes := reservations.MapReservationsHeaderToReservationFieldIndexes(normalizedCsvColumnNames)

	res := make([]sdk.CastaiInventoryV1beta1AzureReservationImport, 0, len(reservationRecords))
	for _, record := range reservationRecords {
		result, err := mapReservationCSVRowToImport(fieldIndexes, record)
		if err != nil {
			return nil, err
		}
		res = append(res, *result)
	}
	return res, nil
}

func mapReservationCSVRowToImport(fieldIndexes map[string]int, record []string) (*sdk.CastaiInventoryV1beta1AzureReservationImport, error) {
	var count *int32
	if countStr := reservations.GetRecordFieldStringValue(reservations.FieldReservationQuantity, fieldIndexes, record); countStr != nil {
		v, err := strconv.Atoi(*countStr)
		if err != nil {
			return nil, fmt.Errorf("parsing quantity: %w", err)
		}
		count = lo.ToPtr(int32(v))
	}

	return &sdk.CastaiInventoryV1beta1AzureReservationImport{
		ExpirationDate:     reservations.GetRecordFieldStringValue(reservations.FieldReservationExpirationDate, fieldIndexes, record),
		Name:               reservations.GetRecordFieldStringValue(reservations.FieldReservationName, fieldIndexes, record),
		ProductName:        reservations.GetRecordFieldStringValue(reservations.FieldReservationProductName, fieldIndexes, record),
		PurchaseDate:       reservations.GetRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		Quantity:           count,
		Region:             reservations.GetRecordFieldStringValue(reservations.FieldReservationRegion, fieldIndexes, record),
		ReservationId:      reservations.GetRecordFieldStringValue(reservations.FieldReservationID, fieldIndexes, record),
		Scope:              reservations.GetRecordFieldStringValue(reservations.FieldReservationPurchaseDate, fieldIndexes, record),
		ScopeResourceGroup: reservations.GetRecordFieldStringValue(reservations.FieldReservationScopeResourceGroup, fieldIndexes, record),
		ScopeSubscription:  reservations.GetRecordFieldStringValue(reservations.FieldReservationScopeSubscription, fieldIndexes, record),
		Status:             reservations.GetRecordFieldStringValue(reservations.FieldReservationStatus, fieldIndexes, record),
		Term:               reservations.GetRecordFieldStringValue(reservations.FieldReservationTerm, fieldIndexes, record),
		Type:               reservations.GetRecordFieldStringValue(reservations.FieldReservationType, fieldIndexes, record),
	}, nil
}

// SortResources sorts the toSort slice based on the order of the targetOrder slice
func SortResources[R Resource](toSort, targetOrder []R) {
	orderMap := make(map[string]int)
	for index, value := range targetOrder {
		orderMap[value.GetIDInCloud()] = index
	}

	slices.SortStableFunc(toSort, func(a, b R) bool {
		indexI, foundI := orderMap[a.GetIDInCloud()]
		indexJ, foundJ := orderMap[b.GetIDInCloud()]

		if !foundI && !foundJ {
			return a.GetIDInCloud() < b.GetIDInCloud()
		}
		if !foundI {
			return true
		}
		if !foundJ {
			return false
		}
		return indexI < indexJ
	})
}
