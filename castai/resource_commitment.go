package castai

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cenkalti/backoff/v4"

	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
)

var (
	_ resource.Resource                   = (*genericCommitmentResource)(nil)
	_ resource.ResourceWithConfigure      = (*genericCommitmentResource)(nil)
	_ resource.ResourceWithImportState    = (*genericCommitmentResource)(nil)
	_ resource.ResourceWithValidateConfig = (*genericCommitmentResource)(nil)
)

// regionRequiredCommitmentTypes lists commitment types that are always
// region-scoped and therefore require the region attribute to be set.
// SAVINGS_PLAN and FLEX_CUD are excluded because they can be region-agnostic.
var regionRequiredCommitmentTypes = map[string]bool{
	string(pricing.CommitmentTypeRESERVEDINSTANCE):            true,
	string(pricing.CommitmentTypeRESOURCECUD):                 true,
	string(pricing.CommitmentTypeCAPACITYBLOCK):               true,
	string(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION): true,
}

// commitmentTypeToDetailsField maps the commitment type enum to the details
// attribute that must be set for that type.
var commitmentTypeToDetailsField = map[string][]string{
	string(pricing.CommitmentTypeRESERVEDINSTANCE):            {"aws_reserved_instances_details", "azure_reservation_details"},
	string(pricing.CommitmentTypeRESOURCECUD):                 {"gcp_resource_cud_details"},
	string(pricing.CommitmentTypeSAVINGSPLAN):                 {"aws_savings_plan_details", "azure_savings_plan_details"},
	string(pricing.CommitmentTypeCAPACITYBLOCK):               {"aws_capacity_block_details"},
	string(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION): {"aws_odcr_details", "gcp_capacity_reservation_details"},
	string(pricing.CommitmentTypeFLEXCUD):                     {"gcp_flex_cud_details"},
}

// detailsFieldToCloud maps each details block attribute name to the cloud
// provider it belongs to. Used to validate that the details block matches
// both the commitment type and the cloud.
var detailsFieldToCloud = map[string]string{
	"aws_reserved_instances_details":   string(pricing.CommitmentCloudAWS),
	"azure_reservation_details":        string(pricing.CommitmentCloudAZURE),
	"gcp_resource_cud_details":         string(pricing.CommitmentCloudGCP),
	"aws_savings_plan_details":         string(pricing.CommitmentCloudAWS),
	"aws_capacity_block_details":       string(pricing.CommitmentCloudAWS),
	"aws_odcr_details":                 string(pricing.CommitmentCloudAWS),
	"gcp_flex_cud_details":             string(pricing.CommitmentCloudGCP),
	"azure_savings_plan_details":       string(pricing.CommitmentCloudAZURE),
	"gcp_capacity_reservation_details": string(pricing.CommitmentCloudGCP),
}

type genericCommitmentResource struct {
	client *ProviderConfig
}

func newCommitmentResource() resource.Resource {
	return &genericCommitmentResource{}
}

func (r *genericCommitmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_commitment"
}

// requiresReplaceString is a shorthand for a string attribute plan modifier list
// that forces resource replacement on change.
func requiresReplaceString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

func (r *genericCommitmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single CAST AI commitment (reserved instance, savings plan, CUD, " +
			"capacity block or capacity reservation) via the pricing v1beta Commitments API. " +
			"For bulk management, use for_each — see the examples/ directory.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Commitment ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "CAST AI organization ID. If not provided, the organization from the " +
					"provider configuration is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Commitment name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"cloud": schema.StringAttribute{
				Required:      true,
				Description:   "Cloud provider. One of: AWS, GCP, AZURE.",
				PlanModifiers: requiresReplaceString(),
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(pricing.CommitmentCloudAWS),
						string(pricing.CommitmentCloudGCP),
						string(pricing.CommitmentCloudAZURE),
					),
				},
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Region where the commitment is applicable. Not required for region-agnostic commitment types such as Compute Savings Plans.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				Description: "Commitment type. One of: RESERVED_INSTANCE, RESOURCE_CUD, SAVINGS_PLAN, " +
					"CAPACITY_BLOCK, ON_DEMAND_CAPACITY_RESERVATION, FLEX_CUD. The matching details block must be set.",
				PlanModifiers: requiresReplaceString(),
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(pricing.CommitmentTypeRESERVEDINSTANCE),
						string(pricing.CommitmentTypeRESOURCECUD),
						string(pricing.CommitmentTypeSAVINGSPLAN),
						string(pricing.CommitmentTypeCAPACITYBLOCK),
						string(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION),
						string(pricing.CommitmentTypeFLEXCUD),
					),
				},
			},
			"start_time": schema.StringAttribute{
				Required:    true,
				Description: "Start time of the commitment (RFC3339, e.g. 2026-01-01T00:00:00Z).",
				Validators:  []validator.String{rfc3339Validator{}},
			},
			"end_time": schema.StringAttribute{
				Optional:    true,
				Description: "End time of the commitment (RFC3339).",
				Validators:  []validator.String{rfc3339Validator{}},
			},
			"autoscaling_status": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(string(pricing.CommitmentAutoscalingStatusINACTIVE)),
				Description: "Controls whether the autoscaler can use this commitment. One of: ACTIVE, INACTIVE. " +
					"Defaults to INACTIVE (matches the API default); set to ACTIVE for the autoscaler to use the commitment.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(pricing.CommitmentAutoscalingStatusACTIVE),
						string(pricing.CommitmentAutoscalingStatusINACTIVE),
					),
				},
			},
			"allowed_usage": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(1.0),
				Description: "Allowed usage of the commitment (0.0-1.0). 1.0 means 100%. Defaults to 1.0.",
				Validators: []validator.Float64{
					float64validator.Between(0.0, 1.0),
				},
			},
			"prioritization": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If enabled, resources used by lower-priority clusters are ignored.",
			},
			"scaling_strategy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(pricing.CommitmentScalingStrategyDEFAULT)),
				Description: "Scaling strategy for the autoscaler. One of: DEFAULT, CPU_BASED, MEMORY_BASED.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(pricing.CommitmentScalingStrategyDEFAULT),
						string(pricing.CommitmentScalingStrategyCPUBASED),
						string(pricing.CommitmentScalingStrategyMEMORYBASED),
					),
				},
			},
			"auto_assignment": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Auto-assign the commitment to all matching clusters in the commitment's region. " +
					"If not set, the server's auto-assignment logic determines the value at creation time. " +
					"Set to false to explicitly disable auto-assignment. " +
					"Not supported for CAPACITY_BLOCK or targeted ON_DEMAND_CAPACITY_RESERVATION commitments.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "Computed lifecycle state of the commitment.",
			},
			"create_time": schema.StringAttribute{
				Computed:    true,
				Description: "Creation time.",
			},
			"update_time": schema.StringAttribute{
				Computed:    true,
				Description: "Last update time.",
			},
			"aws_reserved_instances_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS Reserved Instances details. Required when type is RESERVED_INSTANCE on AWS.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "AWS reserved instance ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"scope": schema.StringAttribute{
						Optional:    true,
						Description: "Scope: whether the commitment applies to a region or an availability zone.",
					},
					"availability_zone_name": schema.StringAttribute{
						Optional:    true,
						Description: "AWS availability zone name (zonal RIs only).",
					},
					"availability_zone_id": schema.StringAttribute{
						Optional:    true,
						Description: "AWS availability zone ID (zonal RIs only).",
					},
					"instance_type": schema.StringAttribute{
						Required:    true,
						Description: "AWS EC2 instance type.",
					},
					"instance_count": schema.Int64Attribute{
						Required:    true,
						Description: "Instance count.",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "Reserved instance state (e.g. active).",
					},
				},
			},
			"azure_reservation_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Azure Reservation details. Required when type is RESERVED_INSTANCE on Azure.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "Reservation ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"plan": schema.StringAttribute{
						Optional:    true,
						Description: "Reservation term plan. One of: ONE_YEAR, THREE_YEAR.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.AzureReservationDetailsPlanONEYEAR),
								string(pricing.AzureReservationDetailsPlanTHREEYEAR),
							),
						},
					},
					"status": schema.StringAttribute{
						Optional:    true,
						Description: "Provisioning state (e.g. Succeeded).",
					},
					"scope": schema.StringAttribute{
						Optional:    true,
						Description: "Scope (e.g. Shared, Single, ManagementGroup).",
					},
					"scope_subscription": schema.StringAttribute{
						Optional:    true,
						Description: "Scope subscription.",
					},
					"scope_resource_group": schema.StringAttribute{
						Optional:    true,
						Description: "Scope resource group.",
					},
					"scope_management_group": schema.StringAttribute{
						Optional:    true,
						Description: "Scope management group.",
					},
					"scope_tenant": schema.StringAttribute{
						Optional:    true,
						Description: "Scope tenant ID.",
					},
					"instance_type": schema.StringAttribute{
						Required:    true,
						Description: "Reserved instance type.",
					},
					"count": schema.Int64Attribute{
						Required:    true,
						Description: "Count of reserved instances.",
						Validators: []validator.Int64{
							int64validator.AtMost(math.MaxInt32),
						},
					},
					"instance_flexibility": schema.StringAttribute{
						Optional:    true,
						Description: "Instance flexibility. One of: ON, OFF.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.ON),
								string(pricing.OFF),
							),
						},
					},
					"purchase_time": schema.StringAttribute{
						Optional:    true,
						Description: "Purchase time (RFC3339).",
						Validators:  []validator.String{rfc3339Validator{}},
					},
				},
			},
			"gcp_resource_cud_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP Resource CUD details. Required when type is RESOURCE_CUD.",
				Attributes: map[string]schema.Attribute{
					"cud_id": schema.StringAttribute{
						Required:      true,
						Description:   "CUD ID from GCP.",
						PlanModifiers: requiresReplaceString(),
					},
					"plan": schema.StringAttribute{
						Optional:    true,
						Description: "CUD plan. One of: TWELVE_MONTH, THIRTY_SIX_MONTH.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.GCPResourceCUDDetailsPlanTWELVEMONTH),
								string(pricing.GCPResourceCUDDetailsPlanTHIRTYSIXMONTH),
							),
						},
					},
					"type": schema.StringAttribute{
						Optional:    true,
						Description: "GCP commitment type (e.g. COMPUTE_OPTIMIZED_C2D).",
					},
					"memory_mb": schema.Int64Attribute{
						Optional:    true,
						Description: "Committed memory in MB.",
					},
					"cpu": schema.Int64Attribute{
						Optional:    true,
						Description: "Committed CPU count.",
					},
					"status": schema.StringAttribute{
						Optional:    true,
						Description: "Commitment status (e.g. ACTIVE).",
					},
				},
			},
			"aws_savings_plan_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS Savings Plan details. Required when type is SAVINGS_PLAN on AWS.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "AWS savings plan ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"offering_id": schema.StringAttribute{
						Optional:    true,
						Description: "AWS savings plan offering ID.",
					},
					"type": schema.StringAttribute{
						Optional:    true,
						Description: "Savings plan type (e.g. Compute, EC2Instance).",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "Savings plan state (e.g. active).",
					},
					"region": schema.StringAttribute{
						Optional:    true,
						Description: "Savings plan region.",
					},
					"instance_type_family": schema.StringAttribute{
						Optional:    true,
						Description: "Instance type family (EC2 instance savings plans).",
					},
					"commitment_amount": schema.Float64Attribute{
						Optional:    true,
						Description: "Commitment amount in USD per hour.",
					},
					"commitment_term": schema.StringAttribute{
						Optional:    true,
						Description: "Commitment term. One of: COMMITMENT_TERM_UNIT_ONE_YEAR, COMMITMENT_TERM_UNIT_THREE_YEARS.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.COMMITMENTTERMUNITONEYEAR),
								string(pricing.COMMITMENTTERMUNITTHREEYEARS),
							),
						},
					},
					"payment_option": schema.StringAttribute{
						Optional:    true,
						Description: "Payment option. One of: ALL_UPFRONT, PARTIAL_UPFRONT, NO_UPFRONT.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.ALLUPFRONT),
								string(pricing.PARTIALUPFRONT),
								string(pricing.NOUPFRONT),
							),
						},
					},
				},
			},
			"aws_capacity_block_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS Capacity Block details. Required when type is CAPACITY_BLOCK.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "Capacity block reservation ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"availability_zone": schema.StringAttribute{
						Optional:    true,
						Description: "Availability zone name.",
					},
					"availability_zone_id": schema.StringAttribute{
						Optional:    true,
						Description: "Availability zone ID.",
					},
					"instance_type": schema.StringAttribute{
						Required:    true,
						Description: "EC2 instance type.",
					},
					"instance_platform": schema.StringAttribute{
						Optional:    true,
						Description: "Instance platform (e.g. Linux/UNIX).",
					},
					"total_instance_count": schema.Int64Attribute{
						Required:    true,
						Description: "Total reserved instance count.",
					},
					"available_instance_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Available (unused) instance count.",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "Capacity block state.",
					},
				},
			},
			"aws_odcr_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS On-Demand Capacity Reservation details. Required when type is ON_DEMAND_CAPACITY_RESERVATION on AWS.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "Capacity reservation ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"availability_zone": schema.StringAttribute{
						Optional:    true,
						Description: "Availability zone name.",
					},
					"availability_zone_id": schema.StringAttribute{
						Optional:    true,
						Description: "Availability zone ID.",
					},
					"instance_type": schema.StringAttribute{
						Required:    true,
						Description: "EC2 instance type.",
					},
					"instance_platform": schema.StringAttribute{
						Optional:    true,
						Description: "Instance platform (e.g. Linux/UNIX).",
					},
					"tenancy": schema.StringAttribute{
						Optional:    true,
						Description: "Tenancy (default, dedicated).",
					},
					"total_instance_count": schema.Int64Attribute{
						Required:    true,
						Description: "Total reserved instance count.",
					},
					"available_instance_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Available (unused) instance count.",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "Capacity reservation state.",
					},
					"end_date_type": schema.StringAttribute{
						Optional:    true,
						Description: "End date type (unlimited, limited).",
					},
					"instance_match_criteria": schema.StringAttribute{
						Optional:    true,
						Description: "Instance match criteria (open, targeted).",
					},
					"interruptible": schema.BoolAttribute{
						Required:    true,
						Description: "Whether this reservation is interruptible. Set to true for interruptible ODCRs, false otherwise.",
					},
				},
			},
			"gcp_flex_cud_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP Flex CUD details. Required when type is FLEX_CUD.",
				Attributes: map[string]schema.Attribute{
					"order_name": schema.StringAttribute{
						Optional:    true,
						Description: "Resource name of the order.",
					},
					"display_name": schema.StringAttribute{
						Optional:    true,
						Description: "User-specified display name.",
					},
					"line_item_id": schema.StringAttribute{
						Required:      true,
						Description:   "Line item ID within the order.",
						PlanModifiers: requiresReplaceString(),
					},
					"offer": schema.StringAttribute{
						Optional:    true,
						Description: "Offer identifier.",
					},
					"region": schema.StringAttribute{
						Optional:    true,
						Description: "Region (can be empty for global Flex CUDs).",
					},
					"commitment_amount": schema.Float64Attribute{
						Optional:    true,
						Description: "Hourly commitment in USD per hour.",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "State (ACTIVE, CANCELED).",
					},
					"plan": schema.StringAttribute{
						Optional:    true,
						Description: "CUD plan. One of: TWELVE_MONTH, THIRTY_SIX_MONTH.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.GCPFlexCUDDetailsPlanTWELVEMONTH),
								string(pricing.GCPFlexCUDDetailsPlanTHIRTYSIXMONTH),
							),
						},
					},
				},
			},
			"azure_savings_plan_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Azure Savings Plan details. Required when type is SAVINGS_PLAN on Azure.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "Savings plan ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"term": schema.StringAttribute{
						Optional:    true,
						Description: "Savings plan term. One of: ONE_YEAR, THREE_YEARS, FIVE_YEARS.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.AzureSavingsPlanDetailsTermONEYEAR),
								string(pricing.AzureSavingsPlanDetailsTermTHREEYEARS),
								string(pricing.AzureSavingsPlanDetailsTermFIVEYEARS),
							),
						},
					},
					"provisioning_state": schema.StringAttribute{
						Optional:    true,
						Description: "Provisioning state (e.g. Succeeded, Expired, Cancelled).",
					},
					"scope": schema.StringAttribute{
						Optional:    true,
						Description: "Applied scope type. One of: SINGLE, SHARED, MANAGEMENT_GROUP.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(pricing.SINGLE),
								string(pricing.SHARED),
								string(pricing.MANAGEMENTGROUP),
							),
						},
					},
					"scope_subscription": schema.StringAttribute{
						Optional:    true,
						Description: "Scope subscription ID.",
					},
					"scope_resource_group": schema.StringAttribute{
						Optional:    true,
						Description: "Scope resource group.",
					},
					"scope_management_group": schema.StringAttribute{
						Optional:    true,
						Description: "Scope management group.",
					},
					"scope_tenant": schema.StringAttribute{
						Optional:    true,
						Description: "Scope tenant ID.",
					},
					"commitment_amount": schema.Float64Attribute{
						Optional:    true,
						Description: "Hourly commitment amount in USD per hour.",
					},
				},
			},
			"gcp_capacity_reservation_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP capacity reservation details. Required when type is ON_DEMAND_CAPACITY_RESERVATION on GCP.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:      true,
						Description:   "Reservation ID.",
						PlanModifiers: requiresReplaceString(),
					},
					"self_link": schema.StringAttribute{
						Optional:    true,
						Description: "Server-defined URL for the reservation (e.g. https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/reservations/my-reservation).",
					},
					"project_id": schema.StringAttribute{
						Optional:    true,
						Description: "GCP project ID that owns the reservation.",
					},
					"zone": schema.StringAttribute{
						Optional:    true,
						Description: "GCP zone of the reservation.",
					},
					"instance_type": schema.StringAttribute{
						Required:    true,
						Description: "Instance type covered by the reservation.",
					},
					"total_instance_count": schema.Int64Attribute{
						Required:    true,
						Description: "Total number of instances reserved.",
					},
					"in_use_instance_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Number of instances currently in use.",
					},
					"assured_instance_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Assured count for the reservation.",
					},
					"state": schema.StringAttribute{
						Optional:    true,
						Description: "Runtime state of the reservation (e.g. READY).",
					},
					"specific_reservation_required": schema.BoolAttribute{
						Required:    true,
						Description: "Whether the reservation requires specific reservation affinity. Set to true for targeted reservations, false for automatic (open) reservations.",
					},
					"min_cpu_platform": schema.StringAttribute{
						Optional:    true,
						Description: "Minimum CPU platform the reserved VMs require (e.g. Intel Cascade Lake).",
					},
					"share_settings": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Reservation sharing settings.",
						Attributes: map[string]schema.Attribute{
							"share_type": schema.StringAttribute{
								Optional: true,
								Description: "Share type. One of: GCP_RESERVATION_SHARE_TYPE_LOCAL, " +
									"GCP_RESERVATION_SHARE_TYPE_SPECIFIC_PROJECTS, GCP_RESERVATION_SHARE_TYPE_ORGANIZATION.",
								Validators: []validator.String{
									stringvalidator.OneOf(
										string(pricing.GCPRESERVATIONSHARETYPELOCAL),
										string(pricing.GCPRESERVATIONSHARETYPESPECIFICPROJECTS),
										string(pricing.GCPRESERVATIONSHARETYPEORGANIZATION),
									),
								},
							},
							"project_ids": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
								Description: "Project IDs this reservation is shared with (SPECIFIC_PROJECTS share type only).",
							},
						},
					},
					"accelerators": schema.ListNestedAttribute{
						Optional:    true,
						Description: "Accelerators (GPUs) reserved VMs must match exactly.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"accelerator_type": schema.StringAttribute{
									Required:    true,
									Description: "Accelerator (GPU) type as the full GCP acceleratorType URL.",
								},
								"accelerator_count": schema.Int64Attribute{
									Required:    true,
									Description: "Number of accelerators of this type per VM.",
								},
							},
						},
					},
					"local_ssds": schema.ListNestedAttribute{
						Optional:    true,
						Description: "Local SSD disks reserved VMs must match exactly.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"interface": schema.StringAttribute{
									Optional: true,
									Description: "Disk interface. One of: GCP_RESERVATION_LOCAL_SSD_INTERFACE_SCSI, " +
										"GCP_RESERVATION_LOCAL_SSD_INTERFACE_NVME.",
									Validators: []validator.String{
										stringvalidator.OneOf(
											string(pricing.GCPRESERVATIONLOCALSSDINTERFACESCSI),
											string(pricing.GCPRESERVATIONLOCALSSDINTERFACENVME),
										),
									},
								},
								"disk_size_gb": schema.Int64Attribute{
									Required:    true,
									Description: "Disk size in GB.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *genericCommitmentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("aws_reserved_instances_details"),
			path.MatchRoot("azure_reservation_details"),
			path.MatchRoot("gcp_resource_cud_details"),
			path.MatchRoot("aws_savings_plan_details"),
			path.MatchRoot("aws_capacity_block_details"),
			path.MatchRoot("aws_odcr_details"),
			path.MatchRoot("gcp_flex_cud_details"),
			path.MatchRoot("azure_savings_plan_details"),
			path.MatchRoot("gcp_capacity_reservation_details"),
		),
	}
}

// ValidateConfig cross-checks that the details block matches the commitment type.
func (r *genericCommitmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config commitmentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Type.IsNull() || config.Type.IsUnknown() {
		return
	}

	setDetails := map[string]bool{
		"aws_reserved_instances_details":   config.AWSReservedInstancesDetails != nil,
		"azure_reservation_details":        config.AzureReservationDetails != nil,
		"gcp_resource_cud_details":         config.GCPResourceCUDDetails != nil,
		"aws_savings_plan_details":         config.AWSSavingsPlanDetails != nil,
		"aws_capacity_block_details":       config.AWSCapacityBlockDetails != nil,
		"aws_odcr_details":                 config.AWSODCRDetails != nil,
		"gcp_flex_cud_details":             config.GCPFlexCUDDetails != nil,
		"azure_savings_plan_details":       config.AzureSavingsPlanDetails != nil,
		"gcp_capacity_reservation_details": config.GCPCapacityReservationDetails != nil,
	}

	allowed, ok := commitmentTypeToDetailsField[config.Type.ValueString()]
	if !ok {
		return // type validator reports invalid enum values
	}

	for field, isSet := range setDetails {
		if !isSet {
			continue
		}
		allowedForType := false
		for _, a := range allowed {
			if a == field {
				allowedForType = true
				break
			}
		}
		if !allowedForType {
			resp.Diagnostics.AddAttributeError(
				path.Root(field),
				"Details block does not match commitment type",
				fmt.Sprintf("commitment type %q requires one of: %s", config.Type.ValueString(), strings.Join(allowed, ", ")),
			)
		}
	}

	// Validate that the details block matches the configured cloud.
	if !config.Cloud.IsNull() && !config.Cloud.IsUnknown() {
		for field, isSet := range setDetails {
			if !isSet {
				continue
			}
			requiredCloud, ok := detailsFieldToCloud[field]
			if ok && config.Cloud.ValueString() != requiredCloud {
				resp.Diagnostics.AddAttributeError(
					path.Root(field),
					"Details block does not match cloud",
					fmt.Sprintf("%q requires cloud %q, but cloud is set to %q", field, requiredCloud, config.Cloud.ValueString()),
				)
			}
		}
	}

	// Validate that end_time is after start_time.
	if !config.StartTime.IsNull() && !config.StartTime.IsUnknown() &&
		!config.EndTime.IsNull() && !config.EndTime.IsUnknown() {
		start, err1 := time.Parse(time.RFC3339, config.StartTime.ValueString())
		end, err2 := time.Parse(time.RFC3339, config.EndTime.ValueString())
		if err1 == nil && err2 == nil && !end.After(start) {
			resp.Diagnostics.AddAttributeError(
				path.Root("end_time"),
				"end_time must be after start_time",
				fmt.Sprintf("end_time %q is not after start_time %q", config.EndTime.ValueString(), config.StartTime.ValueString()),
			)
		}
	}

	// Validate that region is set for commitment types that are always
	// region-scoped. SAVINGS_PLAN and FLEX_CUD can be region-agnostic and
	// are excluded from this check.
	if regionRequiredCommitmentTypes[config.Type.ValueString()] &&
		(config.Region.IsNull() || config.Region.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"region is required for this commitment type",
			fmt.Sprintf("commitment type %q requires a region; it is only optional for SAVINGS_PLAN and FLEX_CUD", config.Type.ValueString()),
		)
	}
}

func (r *genericCommitmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

// resolveOrganizationID returns the organization ID from the model, falling back
// to the provider-level organization (config attribute, env var or single-org lookup).
func (r *genericCommitmentResource) resolveOrganizationID(ctx context.Context, m *commitmentModel) (string, error) {
	if !m.OrganizationID.IsNull() && !m.OrganizationID.IsUnknown() && m.OrganizationID.ValueString() != "" {
		return m.OrganizationID.ValueString(), nil
	}
	return getDefaultOrganizationId(ctx, r.client)
}

func (r *genericCommitmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan commitmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config commitmentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := r.resolveOrganizationID(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve organization ID", err.Error())
		return
	}

	input, diags := plan.toCreateInput(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.createCommitmentWithRetry(ctx, organizationID, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create commitment", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to create commitment",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	created := apiResp.JSON200
	commitmentID := ""
	if created.Id != nil {
		commitmentID = *created.Id
	}

	// PATCH only when the user explicitly set auto_assignment=false, to override
	// the server's enableAutoAssignments flip. If not set or set to true, the
	// POST response is authoritative.
	needsPatch := !config.AutoAssignment.IsNull() && !config.AutoAssignment.IsUnknown() &&
		!config.AutoAssignment.ValueBool()

	if commitmentID != "" && needsPatch {
		patchResp, patchErr := r.updateCommitmentWithRetry(ctx, organizationID, commitmentID, plan.toUpdateInput())
		if patchErr != nil {
			saveCreatedState(ctx, resp, &plan, organizationID, created)
			resp.Diagnostics.AddError("Failed to enforce commitment settings after create", patchErr.Error())
			return
		}
		if patchResp.StatusCode() != http.StatusOK {
			saveCreatedState(ctx, resp, &plan, organizationID, created)
			resp.Diagnostics.AddError(
				"Failed to enforce commitment settings after create",
				fmt.Sprintf("unexpected status code: %d, body: %s", patchResp.StatusCode(), string(patchResp.Body)),
			)
			return
		}
		created = patchResp.JSON200
	}

	saveCreatedState(ctx, resp, &plan, organizationID, created)
}

// saveCreatedState writes the commitment ID and computed fields from the
// API response into Terraform state. Auto-assignment is also synced from
// the API response so that the server-decided value is captured when the
// user did not set it explicitly.
func saveCreatedState(ctx context.Context, resp *resource.CreateResponse, plan *commitmentModel, organizationID string, created *pricing.Commitment) {
	plan.OrganizationID = types.StringValue(organizationID)
	plan.ID = types.StringPointerValue(created.Id)
	if created.State != nil {
		plan.State = types.StringValue(string(*created.State))
	} else {
		plan.State = types.StringNull()
	}
	plan.AutoAssignment = syncBool(plan.AutoAssignment, created.AutoAssignment)
	plan.CreateTime = syncTime(plan.CreateTime, created.CreateTime)
	plan.UpdateTime = syncTime(plan.UpdateTime, created.UpdateTime)
	// Sync detail fields from the API response so that Computed fields
	// (which are unknown in the plan when omitted by the user) are
	// populated with concrete values in state.
	resp.Diagnostics.Append(plan.applyCommitment(ctx, created)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// newCommitmentBackoff creates an exponential backoff capped at 1 minute and
// bound to the supplied context.
func newCommitmentBackoff(ctx context.Context) backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Minute
	return backoff.WithContext(b, ctx)
}

func (r *genericCommitmentResource) getCommitmentWithRetry(ctx context.Context, organizationID, commitmentID string) (*pricing.CommitmentsAPIGetCommitmentResponse, error) {
	b := newCommitmentBackoff(ctx)
	var apiResp *pricing.CommitmentsAPIGetCommitmentResponse
	err := backoff.Retry(func() error {
		resp, err := r.client.pricingClient.CommitmentsAPIGetCommitmentWithResponse(ctx, organizationID, commitmentID)
		if err != nil {
			return err
		}
		// Retry on transient server errors (500, 502, 503, 504).
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("transient server error: status=%d body=%s", resp.StatusCode(), string(resp.Body))
		}
		apiResp = resp
		return nil
	}, b)
	return apiResp, err
}

func (r *genericCommitmentResource) createCommitmentWithRetry(ctx context.Context, organizationID string, input pricing.CreateCommitmentInput) (*pricing.CommitmentsAPICreateCommitmentResponse, error) {
	b := newCommitmentBackoff(ctx)
	var apiResp *pricing.CommitmentsAPICreateCommitmentResponse
	err := backoff.Retry(func() error {
		resp, err := r.client.pricingClient.CommitmentsAPICreateCommitmentWithResponse(ctx, organizationID, input)
		if err != nil {
			return err
		}
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("transient server error: status=%d body=%s", resp.StatusCode(), string(resp.Body))
		}
		apiResp = resp
		return nil
	}, b)
	return apiResp, err
}

func (r *genericCommitmentResource) deleteCommitmentWithRetry(ctx context.Context, organizationID, commitmentID string) (*pricing.CommitmentsAPIDeleteCommitmentResponse, error) {
	b := newCommitmentBackoff(ctx)
	var apiResp *pricing.CommitmentsAPIDeleteCommitmentResponse
	err := backoff.Retry(func() error {
		resp, err := r.client.pricingClient.CommitmentsAPIDeleteCommitmentWithResponse(ctx, organizationID, commitmentID)
		if err != nil {
			return err
		}
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("transient server error: status=%d body=%s", resp.StatusCode(), string(resp.Body))
		}
		apiResp = resp
		return nil
	}, b)
	return apiResp, err
}

func (r *genericCommitmentResource) updateCommitmentWithRetry(ctx context.Context, organizationID, commitmentID string, input pricing.UpdateCommitmentInput) (*pricing.CommitmentsAPIUpdateCommitmentResponse, error) {
	b := newCommitmentBackoff(ctx)
	var apiResp *pricing.CommitmentsAPIUpdateCommitmentResponse
	err := backoff.Retry(func() error {
		resp, err := r.client.pricingClient.CommitmentsAPIUpdateCommitmentWithResponse(ctx, organizationID, commitmentID, input)
		if err != nil {
			return err
		}
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("transient server error: status=%d body=%s", resp.StatusCode(), string(resp.Body))
		}
		apiResp = resp
		return nil
	}, b)
	return apiResp, err
}

func (r *genericCommitmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state commitmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := r.resolveOrganizationID(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve organization ID", err.Error())
		return
	}

	apiResp, err := r.getCommitmentWithRetry(ctx, organizationID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read commitment", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read commitment",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(state.applyCommitment(ctx, apiResp.JSON200)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// upsertPathChanged reports whether any field that flows through the
// CreateCommitment upsert path changed between plan and state.
func upsertPathChanged(plan, state *commitmentModel) bool {
	if !plan.Name.Equal(state.Name) ||
		!plan.Region.Equal(state.Region) ||
		!plan.StartTime.Equal(state.StartTime) ||
		!plan.EndTime.Equal(state.EndTime) {
		return true
	}
	// Any change inside a details block flows through the upsert path.
	// Identifier fields force replacement via plan modifiers, so a diff here
	// is a non-identity detail change (context is merged server-side).
	return !detailsEqual(plan, state)
}

func detailsEqual(a, b *commitmentModel) bool {
	return a.AWSReservedInstancesDetails.equal(b.AWSReservedInstancesDetails) &&
		a.AzureReservationDetails.equal(b.AzureReservationDetails) &&
		a.GCPResourceCUDDetails.equal(b.GCPResourceCUDDetails) &&
		a.AWSSavingsPlanDetails.equal(b.AWSSavingsPlanDetails) &&
		a.AWSCapacityBlockDetails.equal(b.AWSCapacityBlockDetails) &&
		a.AWSODCRDetails.equal(b.AWSODCRDetails) &&
		a.GCPFlexCUDDetails.equal(b.GCPFlexCUDDetails) &&
		a.AzureSavingsPlanDetails.equal(b.AzureSavingsPlanDetails) &&
		a.GCPCapacityReservationDetails.equal(b.GCPCapacityReservationDetails)
}

// patchPathChanged reports whether any operational setting handled by
// UpdateCommitment changed between plan and state.
func patchPathChanged(plan, state *commitmentModel) bool {
	return !plan.AutoscalingStatus.Equal(state.AutoscalingStatus) ||
		!plan.AllowedUsage.Equal(state.AllowedUsage) ||
		!plan.Prioritization.Equal(state.Prioritization) ||
		!plan.ScalingStrategy.Equal(state.ScalingStrategy) ||
		!plan.AutoAssignment.Equal(state.AutoAssignment)
}

func (r *genericCommitmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state commitmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config commitmentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := r.resolveOrganizationID(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve organization ID", err.Error())
		return
	}
	commitmentID := state.ID.ValueString()

	// User explicitly set auto_assignment=false in config.
	autoAssignmentExplicitlyFalse := !config.AutoAssignment.IsNull() && !config.AutoAssignment.IsUnknown() &&
		!config.AutoAssignment.ValueBool()

	// Identity-adjacent fields (name, region, dates, details content) are updated
	// by re-upserting via CreateCommitment; the server matches the existing row on
	// (organization, type, identifier) and preserves the commitment ID.
	upsertChanged := upsertPathChanged(&plan, &state)
	if upsertChanged {
		input, diags := plan.toCreateInput(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiResp, err := r.createCommitmentWithRetry(ctx, organizationID, input)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update commitment", err.Error())
			return
		}
		if apiResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError(
				"Failed to update commitment",
				fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
			)
			return
		}
		if apiResp.JSON200.Id != nil && *apiResp.JSON200.Id != commitmentID {
			resp.Diagnostics.AddError(
				"Commitment identity changed",
				fmt.Sprintf("re-upsert returned a different commitment ID (%s != %s); "+
					"this indicates an identifier field changed without forcing replacement", *apiResp.JSON200.Id, commitmentID),
			)
			return
		}
	}

	// Send PATCH when operational fields changed, or when the upsert path ran
	// and the user explicitly set auto_assignment=false (to counter
	// enableAutoAssignments' potential false→true flip).
	if patchPathChanged(&plan, &state) || (upsertChanged && autoAssignmentExplicitlyFalse) {
		apiResp, err := r.updateCommitmentWithRetry(ctx, organizationID, commitmentID, plan.toUpdateInput())
		if err != nil {
			resp.Diagnostics.AddError("Failed to update commitment settings", err.Error())
			return
		}
		if apiResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError(
				"Failed to update commitment settings",
				fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
			)
			return
		}
	}

	// Read back for authoritative state.
	getResp, err := r.getCommitmentWithRetry(ctx, organizationID, commitmentID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read commitment after update", err.Error())
		return
	}
	if getResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read commitment after update",
			fmt.Sprintf("unexpected status code: %d, body: %s", getResp.StatusCode(), string(getResp.Body)),
		)
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(plan.applyCommitment(ctx, getResp.JSON200)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *genericCommitmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state commitmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := r.resolveOrganizationID(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve organization ID", err.Error())
		return
	}

	apiResp, err := r.deleteCommitmentWithRetry(ctx, organizationID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete commitment", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		return
	}
	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError(
			"Failed to delete commitment",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}
}

func (r *genericCommitmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import formats: <commitment_id> (organization from provider config) or
	// <organization_id>/<commitment_id>.
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError(
				"Invalid import ID format",
				"expected: <commitment_id> or <organization_id>/<commitment_id>",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError(
				"Invalid import ID format",
				"expected: <commitment_id> or <organization_id>/<commitment_id>",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("expected: <commitment_id> or <organization_id>/<commitment_id>, got: %q", req.ID),
		)
	}
}

// rfc3339Validator validates that a string attribute is a valid RFC3339 timestamp.
type rfc3339Validator struct{}

func (v rfc3339Validator) Description(_ context.Context) string {
	return "value must be a valid RFC3339 timestamp (e.g. 2026-01-01T00:00:00Z)"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid RFC3339 timestamp",
			fmt.Sprintf("%q is not a valid RFC3339 timestamp: %s", req.ConfigValue.ValueString(), err),
		)
	}
}
