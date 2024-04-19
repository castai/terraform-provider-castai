package commitments

import (
	"errors"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/reservations"
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type (
	// Terraform SDK's diff setter uses mapstructure under the hood

	GCPCUDResource struct {
		// CAST AI only fields
		ID             *string  `mapstructure:"id,omitempty"`
		AllowedUsage   *float32 `mapstructure:"allowed_usage,omitempty"`
		Prioritization *bool    `mapstructure:"prioritization,omitempty"`
		Status         *string  `mapstructure:"status,omitempty"`

		// Fields from GCP CUDs export JSON
		CUDID          string `mapstructure:"cud_id"`
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
		Id *string `mapstructure:"id"`
	}
)

var (
	// GCPCUDResourceSchema should align with the fields of GCPCUDResource struct
	GCPCUDResourceSchema = map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "",
		},
		"allowed_usage": {
			Type:        schema.TypeFloat,
			Computed:    true,
			Description: "",
		},
		"prioritization": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "",
		},
		"cud_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"cud_status": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"start_timestamp": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"end_timestamp": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"region": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"cpu": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "",
		},
		"memory_mb": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "",
		},
		"plan": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
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

func MapCUDImportToResource(resource sdk.CastaiInventoryV1beta1GCPCommitmentImport) (*GCPCUDResource, error) {
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

	return &GCPCUDResource{
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

func timeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
