package commitments

import (
	"errors"
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
		Prioritization *bool    `mapstructure:"prioritization,omitempty"`
		Status         *string  `mapstructure:"status,omitempty"`
		AllowedUsage   *float32 `mapstructure:"allowed_usage,omitempty"`
	}

	AzureReservationResource struct {
		ID            *string `mapstructure:"id,omitempty"`   // ID of the commitment
		ReservationID string  `mapstructure:"reservation_id"` // ID of the reservation in Azure
	}

	resource interface {
		GetIDInCloud() string
	}
)

var (
	_ resource = (*GCPCUDResource)(nil)
	_ resource = (*AzureReservationResource)(nil)
)

func (r *GCPCUDResource) GetIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.CUDID
}

func (r *AzureReservationResource) GetIDInCloud() string {
	if r == nil {
		return ""
	}
	return r.ReservationID
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

func MapCUDImportToResource(
	resource sdk.CastaiInventoryV1beta1GCPCommitmentImport,
	config *GCPCUDConfigResource,
) (*GCPCUDResource, error) {
	var cpu, memory int
	if resource.Resources != nil {
		for _, res := range *resource.Resources {
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
	if resource.Region != nil {
		_, region = path.Split(*resource.Region)
	}

	res := &GCPCUDResource{
		CUDID:          lo.FromPtr(resource.Id),
		CUDStatus:      lo.FromPtr(resource.Status),
		EndTimestamp:   lo.FromPtr(resource.EndTimestamp),
		StartTimestamp: lo.FromPtr(resource.StartTimestamp),
		Name:           lo.FromPtr(resource.Name),
		Region:         region,
		CPU:            cpu,
		MemoryMb:       memory,
		Plan:           lo.FromPtr(resource.Plan),
		Type:           lo.FromPtr(resource.Type),
	}
	if config != nil {
		res.AllowedUsage = config.AllowedUsage
		res.Prioritization = config.Prioritization
		res.Status = config.Status
	}
	return res, nil
}

// SortResources sorts the toSort slice based on the order of the targetOrder slice
func SortResources[Resource resource](toSort, targetOrder []Resource) {
	orderMap := make(map[string]int)
	for index, value := range targetOrder {
		orderMap[value.GetIDInCloud()] = index
	}

	slices.SortStableFunc(toSort, func(a, b Resource) int {
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
