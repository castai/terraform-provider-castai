package castai

import (
	"errors"
	"fmt"
	"math"
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

	// NOTE: This type needs to be exported for mapstructure's `squash` tag to work properly
	CASTCommitmentFields struct {
		ID              *string                         `mapstructure:"id,omitempty"` // ID of the commitment
		AllowedUsage    *float64                        `mapstructure:"allowed_usage,omitempty"`
		Prioritization  *bool                           `mapstructure:"prioritization,omitempty"`
		Status          *string                         `mapstructure:"status,omitempty"`
		Assignments     []*commitmentAssignmentResource `mapstructure:"assignments,omitempty"`
		ScalingStrategy *string                         `mapstructure:"scaling_strategy,omitempty"`
	}

	gcpCUDResource struct {
		CASTCommitmentFields `mapstructure:",squash"`
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

	azureReservationResource struct {
		CASTCommitmentFields `mapstructure:",squash"`
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

	// commitmentResource is an interface for common management of GCP (gcpCUDResource) and Azure (azureReservationResource) resources
	commitmentResource interface {
		// GetCommitmentID returns the ID of the commitment in CAST AI
		getCommitmentID() string
		// GetIDInCloud returns the ID of the resource in the cloud provider
		getIDInCloud() string
	}

	commitmentConfigResource struct {
		Matcher         []*commitmentConfigMatcherResource `mapstructure:"matcher,omitempty"`
		Prioritization  *bool                              `mapstructure:"prioritization,omitempty"`
		Status          *string                            `mapstructure:"status,omitempty"`
		AllowedUsage    *float64                           `mapstructure:"allowed_usage,omitempty"`
		Assignments     []*commitmentAssignmentResource    `mapstructure:"assignments,omitempty"`
		ScalingStrategy *string                            `mapstructure:"scaling_strategy,omitempty"`
	}
	commitmentConfigMatcherResource struct {
		Name   string  `mapstructure:"name"`
		Type   *string `mapstructure:"type,omitempty"`
		Region *string `mapstructure:"region,omitempty"`
	}
	commitmentAssignmentResource struct {
		ClusterID string `mapstructure:"cluster_id"`
		Priority  *int   `mapstructure:"priority,omitempty"`
	}
)

func (r *commitmentConfigResource) getMatcher() *commitmentConfigMatcherResource {
	if r == nil || len(r.Matcher) == 0 {
		return nil
	}
	return r.Matcher[0]
}

var (
	_ commitmentResource = (*gcpCUDResource)(nil)
	_ commitmentResource = (*azureReservationResource)(nil)
)

func (r *gcpCUDResource) getCommitmentID() string {
	if r == nil || r.ID == nil {
		return ""
	}
	return *r.ID
}

func (r *gcpCUDResource) getIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.CUDID
}

func (r *azureReservationResource) getCommitmentID() string {
	if r == nil || r.ID == nil {
		return ""
	}
	return *r.ID
}

func (r *azureReservationResource) getIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.ReservationID
}

func (m *commitmentConfigMatcherResource) validate() error {
	if m == nil {
		return errors.New("matcher is required")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func mapCommitmentAssignmentsToResources(
	input []sdk.CastaiInventoryV1beta1CommitmentAssignment,
	prioritizationEnabled bool,
) []*commitmentAssignmentResource {
	return lo.Map(input, func(a sdk.CastaiInventoryV1beta1CommitmentAssignment, _ int) *commitmentAssignmentResource {
		res := &commitmentAssignmentResource{ClusterID: lo.FromPtr(a.ClusterId)}
		if prioritizationEnabled && a.Priority != nil {
			res.Priority = lo.ToPtr(int(*a.Priority))
		}
		return res
	})
}

func mapCommitmentToCUDResource(
	c sdk.CastaiInventoryV1beta1Commitment,
	as []sdk.CastaiInventoryV1beta1CommitmentAssignment,
) (*gcpCUDResource, error) {
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

	return &gcpCUDResource{
		CASTCommitmentFields: CASTCommitmentFields{
			ID:              c.Id,
			AllowedUsage:    float32PtrToFloat64Ptr(c.AllowedUsage, 2),
			Prioritization:  c.Prioritization,
			Status:          (*string)(c.Status),
			Assignments:     mapCommitmentAssignmentsToResources(as, lo.FromPtr(c.Prioritization)),
			ScalingStrategy: (*string)(c.ScalingStrategy),
		},
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

func mapCommitmentToReservationResource(
	c sdk.CastaiInventoryV1beta1Commitment,
	as []sdk.CastaiInventoryV1beta1CommitmentAssignment,
) (*azureReservationResource, error) {
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
	return &azureReservationResource{
		CASTCommitmentFields: CASTCommitmentFields{
			ID:              c.Id,
			AllowedUsage:    float32PtrToFloat64Ptr(c.AllowedUsage, 2),
			Prioritization:  c.Prioritization,
			Status:          (*string)(c.Status),
			Assignments:     mapCommitmentAssignmentsToResources(as, lo.FromPtr(c.Prioritization)),
			ScalingStrategy: (*string)(c.ScalingStrategy),
		},
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

func mapCUDImportToResource(
	cudWithCfg *commitmentWithConfig[castaiGCPCommitmentImport],
) (*gcpCUDResource, error) {
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

	res := &gcpCUDResource{
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
		res.Assignments = cudWithCfg.Config.Assignments
		if lo.FromPtr(cudWithCfg.Config.Prioritization) {
			assignPrioritiesToAssignments(res.Assignments)
		}
		res.ScalingStrategy = cudWithCfg.Config.ScalingStrategy
	}
	if res.ScalingStrategy == nil {
		res.ScalingStrategy = lo.ToPtr("Default")
	}
	return res, nil
}

func mapReservationImportToResource(
	cudWithCfg *commitmentWithConfig[castaiAzureReservationImport],
) (*azureReservationResource, error) {
	res := &azureReservationResource{
		Count:              int(lo.FromPtr(cudWithCfg.Commitment.Quantity)),
		ReservationID:      lo.FromPtr(cudWithCfg.Commitment.ReservationId),
		ReservationStatus:  lo.FromPtr(cudWithCfg.Commitment.Status),
		StartTimestamp:     lo.FromPtr(cudWithCfg.Commitment.PurchaseDate),
		EndTimestamp:       lo.FromPtr(cudWithCfg.Commitment.ExpirationDate),
		Name:               lo.FromPtr(cudWithCfg.Commitment.Name),
		Region:             lo.FromPtr(cudWithCfg.Commitment.Region),
		InstanceType:       lo.FromPtr(cudWithCfg.Commitment.ProductName),
		Plan:               lo.FromPtr(cudWithCfg.Commitment.Term),
		Scope:              lo.FromPtr(cudWithCfg.Commitment.Scope),
		ScopeResourceGroup: lo.FromPtr(cudWithCfg.Commitment.ScopeResourceGroup),
		ScopeSubscription:  lo.FromPtr(cudWithCfg.Commitment.ScopeSubscription),
	}

	switch res.Plan { // normalize the values just like CAST AI's API does
	case "P1Y":
		res.Plan = "ONE_YEAR"
	case "P3Y":
		res.Plan = "THREE_YEAR"
	case "ONE_YEAR":
	case "THREE_YEAR":
	default:
		return nil, fmt.Errorf("invalid plan value: %s", res.Plan)
	}

	if cudWithCfg.Config != nil {
		res.AllowedUsage = cudWithCfg.Config.AllowedUsage
		res.Prioritization = cudWithCfg.Config.Prioritization
		res.Status = cudWithCfg.Config.Status
		res.Assignments = cudWithCfg.Config.Assignments
		if lo.FromPtr(cudWithCfg.Config.Prioritization) {
			assignPrioritiesToAssignments(res.Assignments)
		}
		res.ScalingStrategy = cudWithCfg.Config.ScalingStrategy
	}
	if res.ScalingStrategy == nil {
		res.ScalingStrategy = lo.ToPtr("Default")
	}

	return res, nil
}

// commitmentConfigMatcherKey is a utility type for mapping CUDs to their configurations
type commitmentConfigMatcherKey struct {
	name, region, typ string
}

func (k commitmentConfigMatcherKey) String() string {
	return fmt.Sprintf("%s-%s-%s", k.name, k.region, k.typ)
}

// commitment is a common interface for castaiGCPCommitmentImport and sdk.CastaiInventoryV1beta1Commitment
type commitment interface {
	getKey() commitmentConfigMatcherKey
}

type commitmentWithConfig[C commitment] struct {
	Commitment C
	Config     *commitmentConfigResource
}

func mapConfigsToCommitments[C commitment](
	cmts []C,
	configs []*commitmentConfigResource,
) ([]*commitmentWithConfig[C], error) {
	res := make([]*commitmentWithConfig[C], len(cmts))
	cfgKeys := map[commitmentConfigMatcherKey]struct{}{}
	for _, cfg := range configs {
		cfgKey := commitmentConfigMatcherKey{name: cfg.getMatcher().Name} // Name matcher is required, other fields are optional
		if cfg.getMatcher().Region != nil {
			_, cfgKey.region = path.Split(*cfg.getMatcher().Region)
		}
		if cfg.getMatcher().Type != nil {
			cfgKey.typ = *cfg.getMatcher().Type
		}
		if _, ok := cfgKeys[cfgKey]; ok {
			return nil, fmt.Errorf("duplicate configuration for %s", cfgKey)
		}

		cfgKeys[cfgKey] = struct{}{}

		var assigned bool
		for i, cud := range cmts {
			cudKey := cud.getKey()
			// If the configuration doesn't have a field set, it should match any value of that field
			if cfgKey.region == "" {
				cudKey.region = ""
			}
			if cfgKey.typ == "" {
				cudKey.typ = ""
			}
			if cudKey != cfgKey {
				continue
			}

			if assigned {
				return nil, fmt.Errorf("duplicate import for %s", cfgKey.String())
			}
			if res[i] != nil {
				return nil, fmt.Errorf("commitment already assigned to a configuration")
			}
			res[i] = &commitmentWithConfig[C]{Commitment: cud, Config: cfg}
			assigned = true
		}
		if !assigned {
			return nil, errors.New("not all commitment configurations were mapped")
		}
	}

	// Make sure we don't ignore commitments without configurations
	for i, cud := range cmts {
		if res[i] == nil {
			res[i] = &commitmentWithConfig[C]{Commitment: cud}
		}
	}
	return res, nil
}

func mapConfiguredCUDImportsToResources[C interface {
	castaiGCPCommitmentImport | sdk.CastaiInventoryV1beta1GCPCommitmentImport
}](
	cuds []C,
	configs []*commitmentConfigResource,
) ([]*gcpCUDResource, error) {
	if len(configs) > len(cuds) {
		return nil, fmt.Errorf("more configurations than CUDs")
	}

	var cudImports []castaiGCPCommitmentImport
	switch v := any(cuds).(type) {
	case []castaiGCPCommitmentImport:
		cudImports = v
	case []sdk.CastaiInventoryV1beta1GCPCommitmentImport:
		cudImports = make([]castaiGCPCommitmentImport, 0, len(v))
		for _, item := range v {
			cudImports = append(cudImports, castaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: item})
		}
	}

	cudsWithConfigs, err := mapConfigsToCommitments(cudImports, configs)
	if err != nil {
		return nil, err
	}

	res := make([]*gcpCUDResource, 0, len(cudsWithConfigs))
	for _, item := range cudsWithConfigs {
		v, err := mapCUDImportToResource(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

func mapConfiguredReservationImportsToResources[C interface {
	castaiAzureReservationImport | sdk.CastaiInventoryV1beta1AzureReservationImport
}](
	reservations []C,
	configs []*commitmentConfigResource,
) ([]*azureReservationResource, error) {
	if len(configs) > len(reservations) {
		return nil, fmt.Errorf("more configurations than reservations")
	}

	var cudImports []castaiAzureReservationImport
	switch v := any(reservations).(type) {
	case []castaiAzureReservationImport:
		cudImports = v
	case []sdk.CastaiInventoryV1beta1AzureReservationImport:
		cudImports = make([]castaiAzureReservationImport, 0, len(v))
		for _, item := range v {
			cudImports = append(cudImports, castaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: item})
		}
	}

	cudsWithConfigs, err := mapConfigsToCommitments(cudImports, configs)
	if err != nil {
		return nil, err
	}

	res := make([]*azureReservationResource, 0, len(cudsWithConfigs))
	for _, item := range cudsWithConfigs {
		v, err := mapReservationImportToResource(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)

	}
	return res, nil
}

func mapCommitmentImportWithConfigToUpdateRequest(
	c *commitmentWithConfig[castaiCommitment],
) sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody {
	req := sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
		AllowedUsage:    c.Commitment.AllowedUsage,
		Prioritization:  c.Commitment.Prioritization,
		Status:          c.Commitment.Status,
		ScalingStrategy: c.Commitment.ScalingStrategy,
	}
	if c.Config != nil {
		if c.Config.AllowedUsage != nil {
			req.AllowedUsage = lo.ToPtr(float32(*c.Config.AllowedUsage))
		}
		if c.Config.Prioritization != nil {
			req.Prioritization = c.Config.Prioritization
		}
		if c.Config.Status != nil {
			req.Status = (*sdk.CastaiInventoryV1beta1CommitmentStatus)(c.Config.Status)
		}
		if c.Config.ScalingStrategy != nil {
			req.ScalingStrategy = (*sdk.CastaiInventoryV1beta1CommitmentScalingStrategy)(c.Config.ScalingStrategy)
		}
	}
	return req
}

// Azure specific stuff

func mapReservationCSVRowsToImports(csvRecords [][]string) ([]sdk.CastaiInventoryV1beta1AzureReservationImport, error) {
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
		Scope:              reservations.GetRecordFieldStringValue(reservations.FieldReservationScope, fieldIndexes, record),
		ScopeResourceGroup: reservations.GetRecordFieldStringValue(reservations.FieldReservationScopeResourceGroup, fieldIndexes, record),
		ScopeSubscription:  reservations.GetRecordFieldStringValue(reservations.FieldReservationScopeSubscription, fieldIndexes, record),
		Status:             reservations.GetRecordFieldStringValue(reservations.FieldReservationStatus, fieldIndexes, record),
		Term:               reservations.GetRecordFieldStringValue(reservations.FieldReservationTerm, fieldIndexes, record),
		Type:               reservations.GetRecordFieldStringValue(reservations.FieldReservationType, fieldIndexes, record),
	}, nil
}

// sortCommitmentResources sorts the toSort slice based on the order of the targetOrder slice
func sortCommitmentResources[R commitmentResource](toSort, targetOrder []R) {
	orderMap := make(map[string]int)
	for index, value := range targetOrder {
		orderMap[value.getIDInCloud()] = index
	}

	slices.SortStableFunc(toSort, func(a, b R) bool {
		indexI, foundI := orderMap[a.getIDInCloud()]
		indexJ, foundJ := orderMap[b.getIDInCloud()]

		if !foundI && !foundJ {
			return a.getIDInCloud() < b.getIDInCloud()
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

func assignPrioritiesToAssignments(assignments []*commitmentAssignmentResource) {
	for i, a := range assignments {
		a.Priority = lo.ToPtr(i + 1)
	}
}

// castaiGCPCommitmentImport is a wrapper around sdk.CastaiInventoryV1beta1GCPCommitmentImport implementing the cud interface
type castaiGCPCommitmentImport struct {
	sdk.CastaiInventoryV1beta1GCPCommitmentImport
}

var _ commitment = castaiGCPCommitmentImport{}

func (c castaiGCPCommitmentImport) getKey() commitmentConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	return commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
		typ:    lo.FromPtr(c.Type),
	}
}

// castaiAzureReservationImport is a wrapper around sdk.CastaiInventoryV1beta1AzureReservationImport implementing the cud interface
type castaiAzureReservationImport struct {
	sdk.CastaiInventoryV1beta1AzureReservationImport
}

var _ commitment = castaiAzureReservationImport{}

func (c castaiAzureReservationImport) getKey() commitmentConfigMatcherKey {
	return commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
		typ:    lo.FromPtr(c.ProductName),
	}
}

// castaiCommitment is a wrapper around sdk.CastaiInventoryV1beta1Commitment implementing the cud interface
type castaiCommitment struct {
	sdk.CastaiInventoryV1beta1Commitment
}

var _ commitment = castaiCommitment{}

func (c castaiCommitment) getKey() commitmentConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	res := commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
	}
	if c.GcpResourceCudContext != nil {
		res.typ = *c.GcpResourceCudContext.Type
	}
	if c.AzureReservationContext != nil {
		res.typ = *c.AzureReservationContext.InstanceType
	}
	return res
}

func float32PtrToFloat64Ptr(f *float32, prec int) *float64 {
	if f == nil {
		return nil
	}
	mul := math.Pow10(prec)
	return lo.ToPtr(math.Round(float64(*f)*mul) / mul)
}
