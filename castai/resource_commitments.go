package castai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldCommitmentsAzureReservationsCSV = "azure_reservations_csv"
	FieldCommitmentsGCPCUDsJSON          = "gcp_cuds_json"

	FieldCommitmentsAzureReservations = "azure_reservations"
	FieldCommitmentsGCPCUDs           = "gcp_cuds"
	FieldCommitmentsGCPCUDConfigs     = "gcp_cud_configs"
)

func resourceCommitments() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiCommitmentsRead,
		CreateContext: resourceCastaiCommitmentsCreate,
		UpdateContext: resourceCastaiCommitmentsUpdate,
		DeleteContext: resourceCastaiCommitmentsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: commitmentsStateImporter,
		},
		Description: "Commitments represent cloud service provider reserved instances (Azure) and commited use discounts (GCP) that can be used by CAST AI autoscaler.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},
		CustomizeDiff: commitmentsDiff,
		Schema: map[string]*schema.Schema{
			FieldCommitmentsAzureReservationsCSV: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "CSV file containing reservations exported from Azure.",
				ExactlyOneOf: []string{FieldCommitmentsAzureReservationsCSV, FieldCommitmentsGCPCUDsJSON},
			},
			FieldCommitmentsGCPCUDsJSON: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "JSON file containing CUDs exported from GCP.",
				ExactlyOneOf: []string{FieldCommitmentsAzureReservationsCSV, FieldCommitmentsGCPCUDsJSON},
			},
			FieldCommitmentsGCPCUDs: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of GCP CUDs.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
					},
				},
			},
			FieldCommitmentsGCPCUDConfigs: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of GCP CUD configurations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Matcher fields
						"matcher": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Matcher used to map config to a commitment.",
							MinItems:    1,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Name of the commitment to match.",
									},
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Type of the commitment to match. For compute resources, it's the type of the machine.",
									},
									"region": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Region of the commitment to match.",
									},
								},
							},
						},
						// Actual config fields
						"prioritization": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "If enabled, it's possible to assign priorities to the assigned clusters.",
						},
						"status": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Status of the commitment in CAST AI.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"Active", "Inactive"}, false)),
						},
						"allowed_usage": {
							Type:             schema.TypeFloat,
							Optional:         true,
							Description:      "Allowed usage of the commitment. The value is between 0 (0%) and 1 (100%).",
							ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(0, 1)),
						},
					},
				},
			},
			FieldCommitmentsAzureReservations: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of Azure reservations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{},
				},
			},
		},
	}
}

func commitmentsDiff(_ context.Context, diff *schema.ResourceDiff, _ any) error {
	_, reservationsOk, err := getReservationResources(diff)
	if err != nil {
		return err
	}

	cudResources, cudsOk, err := getCUDImportResources(diff)
	if err != nil {
		return err
	}

	if !reservationsOk && !cudsOk {
		return fmt.Errorf("one of 'azure_reservations_csv' or 'gcp_cuds_json' must be set")
	}
	if reservationsOk && cudsOk {
		return fmt.Errorf("either 'azure_reservations_csv' or 'gcp_cuds_json' can be set, not both")
	}

	switch {
	case reservationsOk:
		// TEMPORARY: support for Azure reservations will be added in one of the upcoming PRs
		if err := diff.SetNew(FieldCommitmentsGCPCUDs, nil); err != nil {
			return fmt.Errorf("setting gcp cuds field to nil: %w", err)
		}
		return fmt.Errorf("azure reservations are currently not supported")
	case cudsOk:
		if err := diff.SetNew(FieldCommitmentsAzureReservations, nil); err != nil {
			return fmt.Errorf("setting azure reservations field to nil: %w", err)
		}
		return diff.SetNew(FieldCommitmentsGCPCUDs, cudResources)
	}

	return errors.New("unhandled combination of commitments input")
}

func getCUDImports(tfData resourceProvider) ([]sdk.CastaiInventoryV1beta1GCPCommitmentImport, bool, error) {
	cudsIface, ok := tfData.GetOk(FieldCommitmentsGCPCUDsJSON)
	if !ok {
		return nil, false, nil
	}
	cudsJSONStr, ok := cudsIface.(string)
	if !ok {
		return nil, true, errors.New("expected 'gcp_cuds_json' to be a string")
	}
	cuds, err := unmarshalCUDs(cudsJSONStr)
	if err != nil {
		return nil, true, err
	}
	return cuds, true, nil
}

func getCUDConfigs(tfData resourceProvider) ([]*GCPCUDConfigResource, error) {
	var configs []*GCPCUDConfigResource
	if configsIface, ok := tfData.GetOk(FieldCommitmentsGCPCUDConfigs); ok {
		if err := mapstructure.Decode(configsIface, &configs); err != nil {
			return nil, err
		}
	}
	return configs, nil
}

// getCUDImportResources returns a slice of GCP CUD resources obtained from the input JSON.
func getCUDImportResources(tfData resourceProvider) ([]*GCPCUDResource, bool, error) {
	// Get the CUD JSON input and unmarshal it into a slice of CUD imports
	cuds, cudsOk, err := getCUDImports(tfData)
	if err != nil {
		return nil, cudsOk, err
	}
	if !cudsOk {
		return nil, false, nil
	}

	// Get the CUD configurations and map them to resources
	configs, err := getCUDConfigs(tfData)
	if err != nil {
		return nil, true, err
	}
	if len(configs) > len(cuds) {
		return nil, true, fmt.Errorf("more configurations than CUDs")
	}
	for _, c := range configs {
		if err := c.GetMatcher().Validate(); err != nil {
			return nil, true, fmt.Errorf("invalid config: %w", err)
		}
	}

	// Finally map the CUD imports to resources and combine them with the configurations
	res, err := MapConfiguredCUDImportsToResources(cuds, configs)
	if err != nil {
		return nil, true, err
	}
	return res, true, nil
}

// getCUDResources returns a slice of GCP CUD resources obtained from the state obtained from the API.
func getCUDResources(tfData resourceProvider) ([]*GCPCUDResource, bool, error) {
	cudsIface, ok := tfData.GetOk(FieldCommitmentsGCPCUDs)
	if !ok {
		return nil, false, nil
	}
	var res []*GCPCUDResource
	if err := mapstructure.Decode(cudsIface, &res); err != nil {
		return nil, true, err
	}
	return res, true, nil
}

func getReservationResources(tfData resourceProvider) ([]*AzureReservationResource, bool, error) {
	// TEMPORARY: support for Azure reservations will be added in one of the upcoming PRs
	_, ok := tfData.GetOk(FieldCommitmentsAzureReservationsCSV)
	return nil, ok, nil
}

func commitmentsStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	if err := populateCommitmentsResourceData(ctx, d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceCastaiCommitmentsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Info(ctx, "Get commitments call start")
	defer tflog.Info(ctx, "Get commitments call end")

	if err := populateCommitmentsResourceData(ctx, d, meta); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceCastaiCommitmentsDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	importID, err := getCommitmentsImportID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	reservations, reservationsOk, err := getReservationResources(data)
	if err != nil {
		return diag.FromErr(err)
	}
	cuds, cudsOk, err := getCUDResources(data)
	if err != nil {
		return diag.FromErr(err)
	}

	switch {
	case reservationsOk:
		if err := deleteCommitments(ctx, meta, reservations); err != nil {
			return diag.FromErr(err)
		}
	case cudsOk:
		if err := deleteCommitments(ctx, meta, cuds); err != nil {
			return diag.FromErr(err)
		}
	}

	data.SetId(importID)
	return nil
}

func deleteCommitments[R Resource](ctx context.Context, meta any, resources []R) error {
	for _, r := range resources {
		if err := deleteCommitment(ctx, meta, r.GetCommitmentID()); err != nil {
			return err
		}
	}
	return nil
}

func resourceCastaiCommitmentsUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Info(ctx, "Update commitments call start")
	defer tflog.Info(ctx, "Update commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Info(ctx, "Create commitments call start")
	defer tflog.Info(ctx, "Create commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsUpsert(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	importID, err := getCommitmentsImportID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, reservationsOk := data.GetOk(FieldCommitmentsAzureReservationsCSV)
	cuds, cudsOk, err := getCUDImports(data)
	if err != nil {
		return diag.FromErr(err)
	}

	switch {
	case reservationsOk:
		return diag.Errorf("azure reservations are currently not supported")
	case cudsOk:
		if err := importCUDs(ctx, meta, cuds); err != nil {
			return diag.FromErr(err)
		}

		orgCommitments, err := getOrganizationCommitments(ctx, meta)
		if err != nil {
			return diag.FromErr(err)
		}
		gcpCommitments := lo.Filter(orgCommitments, func(c sdk.CastaiInventoryV1beta1Commitment, _ int) bool {
			return c.GcpResourceCudContext != nil
		})
		if len(gcpCommitments) != len(cuds) {
			return diag.Errorf("expected %d GCP commitments, got %d", len(cuds), len(gcpCommitments))
		}

		configs, err := getCUDConfigs(data)
		if err != nil {
			return diag.FromErr(err)
		}

		cudsWithConfigs, err := MapConfigsToCUDs(
			lo.Map(gcpCommitments, func(item sdk.CastaiInventoryV1beta1Commitment, _ int) CastaiCommitment {
				return CastaiCommitment{CastaiInventoryV1beta1Commitment: item}
			}),
			configs,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, c := range cudsWithConfigs {
			res, err := meta.(*ProviderConfig).api.CommitmentsAPIUpdateCommitmentWithResponse(
				ctx,
				lo.FromPtr(c.CUD.Id),
				MapCUDImportWithConfigToUpdateRequest(c),
			)
			if err := sdk.CheckOKResponse(res, err); err != nil {
				return diag.Errorf("updating commitment: %v", err)
			}
		}
	}

	data.SetId(importID)
	return resourceCastaiCommitmentsRead(ctx, data, meta)
}

func unmarshalCUDs(input string) (res []sdk.CastaiInventoryV1beta1GCPCommitmentImport, err error) {
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		return nil, err
	}
	return
}

func importCUDs(ctx context.Context, meta any, imports []sdk.CastaiInventoryV1beta1GCPCommitmentImport) error {
	res, err := meta.(*ProviderConfig).api.CommitmentsAPIImportGCPCommitmentsWithResponse(
		ctx,
		&sdk.CommitmentsAPIImportGCPCommitmentsParams{
			Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportGCPCommitmentsParamsBehaviour]("OVERWRITE"),
		},
		imports,
	)
	if checkErr := sdk.CheckOKResponse(res, err); checkErr != nil {
		return fmt.Errorf("upserting commitments: %w", checkErr)
	}
	return nil
}

func deleteCommitment(ctx context.Context, meta any, id string) error {
	tflog.Info(ctx, "Delete commitments call start")
	defer tflog.Info(ctx, "Delete commitments call end")

	res, err := meta.(*ProviderConfig).api.CommitmentsAPIDeleteCommitmentWithResponse(ctx, id)
	if checkErr := sdk.CheckOKResponse(res, err); checkErr != nil {
		return fmt.Errorf("deleting commitments: %w", checkErr)
	}
	return nil
}

func populateCommitmentsResourceData(ctx context.Context, d *schema.ResourceData, meta any) error {
	orgCommitments, err := getOrganizationCommitments(ctx, meta)
	if err != nil {
		return err
	}

	cuds, cudsOk, err := getCUDImportResources(d)
	if err != nil {
		return err
	}

	var resources []*GCPCUDResource
	for _, c := range orgCommitments {
		c := c
		if c.GcpResourceCudContext == nil {
			continue
		}

		resource, err := MapCommitmentToCUDResource(c)
		if err != nil {
			return err
		}
		resources = append(resources, resource)
	}
	if cudsOk {
		SortResources(resources, cuds)
	}
	if err := d.Set(FieldCommitmentsGCPCUDs, resources); err != nil {
		return fmt.Errorf("setting gcp cuds: %w", err)
	}

	return nil
}

func getOrganizationCommitments(ctx context.Context, meta any) ([]sdk.CastaiInventoryV1beta1Commitment, error) {
	client := meta.(*ProviderConfig).api
	response, err := client.CommitmentsAPIGetCommitmentsWithResponse(ctx, &sdk.CommitmentsAPIGetCommitmentsParams{})
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return nil, fmt.Errorf("fetching commitments: %w", checkErr)
	}
	if response.JSON200.Commitments == nil {
		return nil, nil
	}
	return *response.JSON200.Commitments, nil
}

func getCommitmentsImportID(ctx context.Context, data *schema.ResourceData, meta any) (string, error) {
	// The commitments API doesn't take organization ID as a parameter, so we always use the default one associated
	// with the used auth token
	defOrgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return "", err
	}

	var cloud string
	if _, ok := data.GetOk(FieldCommitmentsAzureReservationsCSV); ok {
		cloud = "azure"
	}
	if _, ok := data.GetOk(FieldCommitmentsGCPCUDsJSON); ok {
		cloud = "gcp"
	}
	return defOrgID + ":" + cloud, nil
}
