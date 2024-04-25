package commitments

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"

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
	GCPCUDConfigResource struct {
		// Matcher fields - normally we'd use a struct but Terraform poorly handles nested objects
		MatchName   string  `mapstructure:"match_name"`
		MatchType   *string `mapstructure:"match_type,omitempty"`
		MatchRegion *string `mapstructure:"match_region,omitempty"`
		// Actual config fields
		Prioritization *bool    `mapstructure:"prioritization,omitempty"`
		Status         *string  `mapstructure:"status,omitempty"`
		AllowedUsage   *float32 `mapstructure:"allowed_usage,omitempty"`
	}

	AzureReservationResource struct {
		ID            *string `mapstructure:"id,omitempty"`   // ID of the commitment
		ReservationID string  `mapstructure:"reservation_id"` // ID of the reservation in Azure
	}

	// Resource is an interface for common management of GCP (GCPCUDResource) and Azure (AzureReservationResource) resources
	Resource interface {
		// GetCommitmentID returns the ID of the commitment in CAST AI
		GetCommitmentID() string
		// GetIDInCloud returns the ID of the resource in the cloud provider
		GetIDInCloud() string
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

func (m GCPCUDConfigResource) Validate() error {
	if m.MatchName == "" {
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

func MapCUDImportToResource(
	cudWithCfg *cudWithConfig[CastaiGCPCommitmentImport],
) (*GCPCUDResource, error) {
	var cpu, memory int
	if cudWithCfg.CUD.Resources != nil {
		for _, res := range *cudWithCfg.CUD.Resources {
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
	if cudWithCfg.CUD.Region != nil {
		_, region = path.Split(*cudWithCfg.CUD.Region)
	}

	res := &GCPCUDResource{
		CUDID:          lo.FromPtr(cudWithCfg.CUD.Id),
		CUDStatus:      lo.FromPtr(cudWithCfg.CUD.Status),
		EndTimestamp:   lo.FromPtr(cudWithCfg.CUD.EndTimestamp),
		StartTimestamp: lo.FromPtr(cudWithCfg.CUD.StartTimestamp),
		Name:           lo.FromPtr(cudWithCfg.CUD.Name),
		Region:         region,
		CPU:            cpu,
		MemoryMb:       memory,
		Plan:           lo.FromPtr(cudWithCfg.CUD.Plan),
		Type:           lo.FromPtr(cudWithCfg.CUD.Type),
	}
	if cudWithCfg.Config != nil {
		res.AllowedUsage = cudWithCfg.Config.AllowedUsage
		res.Prioritization = cudWithCfg.Config.Prioritization
		res.Status = cudWithCfg.Config.Status
	}
	return res, nil
}

// cudConfigMatcherKey is a utility type for mapping CUDs to their configurations
type cudConfigMatcherKey struct {
	name, region, typ string
}

func (k cudConfigMatcherKey) String() string {
	return fmt.Sprintf("%s-%s-%s", k.name, k.region, k.typ)
}

// cud is a common interface for CastaiGCPCommitmentImport and sdk.CastaiInventoryV1beta1Commitment
type cud interface {
	getCUDKey() cudConfigMatcherKey
}

type cudWithConfig[C cud] struct {
	CUD    C
	Config *GCPCUDConfigResource
}

func MapConfigsToCUDs[C cud](cuds []C, configs []*GCPCUDConfigResource) ([]*cudWithConfig[C], error) {
	res := make([]*cudWithConfig[C], 0, len(cuds))
	configsByKey := map[cudConfigMatcherKey]*GCPCUDConfigResource{}
	for _, c := range configs {
		var region string
		if c.MatchRegion != nil {
			_, region = path.Split(*c.MatchRegion)
		}
		key := cudConfigMatcherKey{
			name:   c.MatchName,
			region: region,
			typ:    lo.FromPtr(c.MatchType),
		}
		if _, ok := configsByKey[key]; ok { // Make sure each config matcher is unique
			return nil, fmt.Errorf("duplicate CUD configuration for %s", key.String())
		}
		configsByKey[key] = c
	}

	var mappedConfigs int
	processedCUDKeys := map[cudConfigMatcherKey]struct{}{}
	for _, cud := range cuds {
		key := cud.getCUDKey()
		if _, ok := processedCUDKeys[key]; ok { // Make sure each CUD is unique
			return nil, fmt.Errorf("duplicate CUD import for %s", key.String())
		}
		processedCUDKeys[key] = struct{}{}

		config, hasConfig := configsByKey[key]
		if hasConfig {
			mappedConfigs++
		}
		res = append(res, &cudWithConfig[C]{
			CUD:    cud,
			Config: config,
		})
	}
	if mappedConfigs != len(configs) { // Make sure all configs were mapped
		return nil, fmt.Errorf("not all CUD configurations were mapped")
	}
	return res, nil
}

func MapConfiguredCUDImportsToResources[C interface {
	CastaiGCPCommitmentImport | sdk.CastaiInventoryV1beta1GCPCommitmentImport
}](
	cuds []C,
	configs []*GCPCUDConfigResource,
) ([]*GCPCUDResource, error) {
	if len(configs) > len(cuds) {
		return nil, fmt.Errorf("more CUD configurations than CUDs")
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

	cudsWithConfigs, err := MapConfigsToCUDs(cudImports, configs)
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

func MapCUDImportWithConfigToUpdateRequest(
	c *cudWithConfig[CastaiCommitment],
) sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody {
	req := sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
		AllowedUsage:          c.CUD.AllowedUsage,
		EndDate:               c.CUD.EndDate,
		GcpResourceCudContext: c.CUD.GcpResourceCudContext,
		Id:                    c.CUD.Id,
		Name:                  c.CUD.Name,
		Prioritization:        c.CUD.Prioritization,
		Region:                c.CUD.Region,
		StartDate:             c.CUD.StartDate,
		Status:                c.CUD.Status,
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
