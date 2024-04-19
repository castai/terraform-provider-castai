package commitments

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"

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
		Matcher        GCPCUDConfigMatcherResource `mapstructure:"matcher"`
		Prioritization *bool                       `mapstructure:"prioritization,omitempty"`
		Status         *string                     `mapstructure:"status,omitempty"`
		AllowedUsage   *float32                    `mapstructure:"allowed_usage,omitempty"`
	}
	GCPCUDConfigMatcherResource struct {
		Name   string  `mapstructure:"name"`
		Type   *string `mapstructure:"type,omitempty"`
		Region *string `mapstructure:"region,omitempty"`
	}

	AzureReservationResource struct {
		ID            *string `mapstructure:"id,omitempty"`   // ID of the commitment
		ReservationID string  `mapstructure:"reservation_id"` // ID of the reservation in Azure
	}

	Resource interface {
		GetCommitmentID() string
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

func (m GCPCUDConfigMatcherResource) Validate() error {
	if m.Name == "" {
		return errors.New("matcher name is required")
	}
	return nil
}

var (
	// GCPCUDResourceSchema should align with the fields of GCPCUDResource struct
	GCPCUDResourceSchema = map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of the commitment in CAST AI.",
		},
		"allowed_usage": {
			Type:        schema.TypeFloat,
			Computed:    true,
			Description: "Allowed usage of the commitment. The value is between 0 (0%) and 1 (100%).",
		},
		"prioritization": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "If enabled, it's possible to assign priorities to the assigned clusters.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Status of the commitment in CAST AI.",
		},
		"cud_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "ID of the CUD in GCP.",
		},
		"cud_status": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Status of the CUD in GCP.",
		},
		"start_timestamp": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Start timestamp of the CUD.",
		},
		"end_timestamp": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "End timestamp of the CUD.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the CUD.",
		},
		"region": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Region in which the CUD is available.",
		},
		"cpu": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Number of CPUs covered by the CUD.",
		},
		"memory_mb": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Amount of memory in MB covered by the CUD.",
		},
		"plan": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "CUD plan e.g. 'TWELVE_MONTH'.",
		},
		"type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Type of the CUD, e.g. determines the covered resource type e.g. 'COMPUTE_OPTIMIZED_C2D'.",
		},
	}

	AzureReservationResourceSchema = map[string]*schema.Schema{}
)

func MapCommitmentToCUDResource(c sdk.CastaiInventoryV1beta1Commitment) (*GCPCUDResource, error) {
	if c.GcpResourceCudContext == nil {
		return nil, errors.New("missing GCP resource CUD context")
	}

	var cpu, memory int
	if c.GcpResourceCudContext != nil {
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
	}

	return &GCPCUDResource{
		ID:             c.Id,
		AllowedUsage:   c.AllowedUsage,
		Prioritization: c.Prioritization,
		Status:         (*string)(c.Status),
		CUDID:          lo.FromPtr(c.GcpResourceCudContext.CudId),
		CUDStatus:      lo.FromPtr(c.GcpResourceCudContext.Status),
		EndTimestamp:   timeToString(c.EndDate),
		StartTimestamp: timeToString(c.StartDate),
		Name:           lo.FromPtr(c.Name),
		Region:         lo.FromPtr(c.Region),
		CPU:            cpu,
		MemoryMb:       memory,
		Plan:           lo.FromPtr((*string)(c.GcpResourceCudContext.Plan)),
		Type:           lo.FromPtr(c.GcpResourceCudContext.Type),
	}, nil
}

type CUDImport struct {
	sdk.CastaiInventoryV1beta1GCPCommitmentImport
}

func (c CUDImport) getCUDKey() cudConfigMatcherKey {
	return cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
		typ:    lo.FromPtr(c.Type),
	}
}

var _ cud = CUDImport{}

type CUD struct {
	sdk.CastaiInventoryV1beta1Commitment
}

func (c CUD) getCUDKey() cudConfigMatcherKey {
	res := cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
	}
	if c.GcpResourceCudContext != nil {
		res.typ = *c.GcpResourceCudContext.Type
	}
	return res
}

var _ cud = CUD{}

func MapCUDImportToResource(
	cudWithCfg *cudImportWithConfig[CUDImport],
) (*GCPCUDResource, error) {
	var cpu, memory int
	if cudWithCfg.CUD.Resources != nil {
		for _, res := range *cudWithCfg.CUD.Resources {
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

// SortResources sorts the toSort slice based on the order of the targetOrder slice
func SortResources[R Resource](toSort, targetOrder []R) {
	orderMap := make(map[string]int)
	for index, value := range targetOrder {
		orderMap[value.GetIDInCloud()] = index
	}

	slices.SortStableFunc(toSort, func(a, b R) int {
		indexI, foundI := orderMap[a.GetIDInCloud()]
		indexJ, foundJ := orderMap[b.GetIDInCloud()]

		if !foundI && !foundJ {
			if a.GetIDInCloud() < b.GetIDInCloud() {
				return -1
			}
			return 1
		}
		if !foundI {
			return 1
		}
		if !foundJ {
			return -1
		}
		if indexI < indexJ {
			return -1
		}
		return 1
	})
}

func timeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

type cudConfigMatcherKey struct {
	name, region, typ string
}

func (k cudConfigMatcherKey) String() string {
	return fmt.Sprintf("%s-%s-%s", k.name, k.region, k.typ)
}

type cud interface {
	getCUDKey() cudConfigMatcherKey
}

type cudImportWithConfig[C cud] struct {
	CUD    C
	Config *GCPCUDConfigResource
}

func MapConfigsToCUDs[C cud](
	cuds []C,
	configs []*GCPCUDConfigResource,
) ([]*cudImportWithConfig[C], error) {
	res := make([]*cudImportWithConfig[C], 0, len(cuds))
	configsByKey := map[cudConfigMatcherKey]*GCPCUDConfigResource{}
	for _, c := range configs {
		key := cudConfigMatcherKey{
			name:   c.Matcher.Name,
			region: lo.FromPtr(c.Matcher.Region),
			typ:    lo.FromPtr(c.Matcher.Type),
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
		res = append(res, &cudImportWithConfig[C]{
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
	CUDImport | sdk.CastaiInventoryV1beta1GCPCommitmentImport
}](
	cuds []C,
	configs []*GCPCUDConfigResource,
) ([]*GCPCUDResource, error) {
	if len(configs) > len(cuds) {
		return nil, fmt.Errorf("more CUD configurations than CUDs")
	}

	var cudImports []CUDImport
	switch v := any(cuds).(type) {
	case []CUDImport:
		cudImports = v
	case []sdk.CastaiInventoryV1beta1GCPCommitmentImport:
		cudImports = make([]CUDImport, 0, len(v))
		for _, item := range v {
			cudImports = append(cudImports, CUDImport{CastaiInventoryV1beta1GCPCommitmentImport: item})
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

func CUDWithConfigToUpdateCommitmentRequest(
	c *cudImportWithConfig[CUD],
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
