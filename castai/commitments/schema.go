package commitments

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

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
	GCPCUDConfigsSchema = map[string]*schema.Schema{
		// Matcher fields
		"match_name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the commitment to match.",
		},
		"match_type": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Type of the commitment to match. For compute resources, it's the type of the machine.",
		},
		"match_region": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Region of the commitment to match.",
		},
		// Actual config fields
		"prioritization": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: GCPCUDResourceSchema["prioritization"].Description,
		},
		"status": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      GCPCUDResourceSchema["status"].Description,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"Active", "Inactive"}, false)),
		},
		"allowed_usage": {
			Type:             schema.TypeFloat,
			Optional:         true,
			Description:      GCPCUDResourceSchema["allowed_usage"].Description,
			ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(0, 1)),
		},
	}

	AzureReservationResourceSchema = map[string]*schema.Schema{}
)
