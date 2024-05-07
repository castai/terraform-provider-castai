package castai

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldCommitmentsAzureReservationsCSV = "azure_reservations_csv"
	FieldCommitmentsGCPCUDsJSON          = "gcp_cuds_json"

	FieldCommitmentsAzureReservations = "azure_reservations"
	FieldCommitmentsGCPCUDs           = "gcp_cuds"
	FieldCommitmentsConfigs           = "commitment_configs"
)

var (
	sharedCommitmentResourceSchema = lo.Assign(assignmentsSchema, map[string]*schema.Schema{
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
	})

	assignmentsSchema = map[string]*schema.Schema{
		"assignments": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of assigned clusters for the commitment. If prioritization is enabled, the order of the assignments indicates the priority. The first assignment has the highest priority.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cluster_id": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "ID of the cluster to assign the commitment to.",
					},
					"priority": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Priority of the assignment. The lower the value, the higher the priority. 1 is the highest priority.",
					},
				},
			},
		},
	}
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
			// Input files
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
			// Input configurations
			FieldCommitmentsConfigs: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of commitment configurations.",
				Elem: &schema.Resource{
					Schema: lo.Assign(assignmentsSchema, map[string]*schema.Schema{
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
					}),
				},
			},
			// Computed fields
			FieldCommitmentsGCPCUDs: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of GCP CUDs.",
				Elem: &schema.Resource{
					Schema: lo.Assign(sharedCommitmentResourceSchema, map[string]*schema.Schema{
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
					}),
				},
			},
			FieldCommitmentsAzureReservations: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of Azure reservations.",
				Elem: &schema.Resource{
					Schema: lo.Assign(sharedCommitmentResourceSchema, map[string]*schema.Schema{
						"count": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Number of instances covered by the reservation.",
						},
						"reservation_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the reservation in Azure.",
						},
						"instance_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Type of the instance covered by the reservation.",
						},
						"plan": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Plan of the reservation.",
						},
						"scope": {
							Type:     schema.TypeString,
							Required: true,
						},
						"scope_resource_group": {
							Type:     schema.TypeString,
							Required: true,
						},
						"scope_subscription": {
							Type:     schema.TypeString,
							Required: true,
						},
						"reservation_status": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Status of the reservation in Azure.",
						},
					}),
				},
			},
		},
	}
}

func commitmentsDiff(_ context.Context, diff *schema.ResourceDiff, _ any) error {
	reservationResources, reservationsOk, err := getReservationImportResources(diff)
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
		return diff.SetNew(FieldCommitmentsAzureReservations, reservationResources)
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

func getReservationImports(tfData resourceProvider) ([]sdk.CastaiInventoryV1beta1AzureReservationImport, bool, error) {
	reservationsIface, ok := tfData.GetOk(FieldCommitmentsAzureReservationsCSV)
	if !ok {
		return nil, false, nil
	}
	reservationsCSVStr, ok := reservationsIface.(string)
	if !ok {
		return nil, true, errors.New("expected 'azure_reservations_csv' to be a string")
	}

	csvReader := csv.NewReader(strings.NewReader(reservationsCSVStr))
	csvRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, true, fmt.Errorf("parsing reservations csv: %w", err)
	}

	resources, err := MapReservationCSVRowsToImports(csvRecords)
	if err != nil {
		return nil, true, err
	}
	return resources, true, nil
}

func getCommitmentConfigs(tfData resourceProvider) ([]*CommitmentConfigResource, error) {
	var configs []*CommitmentConfigResource
	if configsIface, ok := tfData.GetOk(FieldCommitmentsConfigs); ok {
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

	// Get the configurations and map them to resources
	configs, err := getCommitmentConfigs(tfData)
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

func getReservationImportResources(tfData resourceProvider) ([]*AzureReservationResource, bool, error) {
	reservations, reservationsOk, err := getReservationImports(tfData)
	if err != nil {
		return nil, reservationsOk, err
	}
	if !reservationsOk {
		return nil, false, nil
	}

	// Get the configurations and map them to resources
	configs, err := getCommitmentConfigs(tfData)
	if err != nil {
		return nil, true, err
	}
	if len(configs) > len(reservations) {
		return nil, true, fmt.Errorf("more configurations than reservations")
	}
	for _, c := range configs {
		if err := c.GetMatcher().Validate(); err != nil {
			return nil, true, fmt.Errorf("invalid config: %w", err)
		}
	}

	// Finally map the reservation imports to resources and combine them with the configurations
	res, err := MapConfiguredReservationImportsToResources(reservations, configs)
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
	reservationsIface, ok := tfData.GetOk(FieldCommitmentsAzureReservations)
	if !ok {
		return nil, false, nil
	}
	var res []*AzureReservationResource
	if err := mapstructure.Decode(reservationsIface, &res); err != nil {
		return nil, true, err
	}
	return res, true, nil
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

	reservations, reservationsOk, err := getReservationImports(data)
	if err != nil {
		return diag.FromErr(err)
	}
	cuds, cudsOk, err := getCUDImports(data)
	if err != nil {
		return diag.FromErr(err)
	}

	var imported []sdk.CastaiInventoryV1beta1Commitment
	switch {
	case reservationsOk:
		if err := importReservations(ctx, meta, reservations); err != nil {
			return diag.FromErr(err)
		}
		orgCommitments, err := getOrganizationCommitments(ctx, meta)
		if err != nil {
			return diag.FromErr(err)
		}
		imported = lo.Filter(orgCommitments, func(c sdk.CastaiInventoryV1beta1Commitment, _ int) bool {
			return c.AzureReservationContext != nil
		})
		if len(imported) != len(reservations) {
			return diag.Errorf("expected %d Azure commitments, got %d", len(reservations), len(imported))
		}
	case cudsOk:
		if err := importCUDs(ctx, meta, cuds); err != nil {
			return diag.FromErr(err)
		}
		orgCommitments, err := getOrganizationCommitments(ctx, meta)
		if err != nil {
			return diag.FromErr(err)
		}
		imported = lo.Filter(orgCommitments, func(c sdk.CastaiInventoryV1beta1Commitment, _ int) bool {
			return c.GcpResourceCudContext != nil
		})
		if len(imported) != len(cuds) {
			return diag.Errorf("expected %d GCP commitments, got %d", len(cuds), len(imported))
		}
	}

	configs, err := getCommitmentConfigs(data)
	if err != nil {
		return diag.FromErr(err)
	}

	cudsWithConfigs, err := MapConfigsToCommitments(
		lo.Map(imported, func(item sdk.CastaiInventoryV1beta1Commitment, _ int) CastaiCommitment {
			return CastaiCommitment{CastaiInventoryV1beta1Commitment: item}
		}),
		configs,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	client := meta.(*ProviderConfig).api
	for _, c := range cudsWithConfigs {
		commitmentID := lo.FromPtr(c.Commitment.Id)
		res, err := client.CommitmentsAPIUpdateCommitmentWithResponse(
			ctx,
			commitmentID,
			MapCommitmentImportWithConfigToUpdateRequest(c),
		)
		if err := sdk.CheckOKResponse(res, err); err != nil {
			return diag.Errorf("updating commitment: %v", err)
		}

		var clusterIDs []string
		if c.Config != nil {
			clusterIDs = lo.Map(c.Config.Assignments, func(a *CommitmentAssignmentResource, _ int) string {
				return a.ClusterID
			})
		}
		asRes, err := client.CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(ctx, commitmentID, clusterIDs)
		if err := sdk.CheckOKResponse(asRes, err); err != nil {
			return diag.Errorf("replacing commitment assignments: %v", err)
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
		return fmt.Errorf("importing gcp cuds: %w", checkErr)
	}
	return nil
}

func importReservations(ctx context.Context, meta any, imports []sdk.CastaiInventoryV1beta1AzureReservationImport) error {
	res, err := meta.(*ProviderConfig).api.CommitmentsAPIImportAzureReservationsWithResponse(
		ctx,
		&sdk.CommitmentsAPIImportAzureReservationsParams{
			Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportAzureReservationsParamsBehaviour]("OVERWRITE"),
		},
		imports,
	)
	if checkErr := sdk.CheckOKResponse(res, err); checkErr != nil {
		return fmt.Errorf("importing azure reservations: %w", checkErr)
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
	// schema.ResourceData contains a blank state instance when the function is called by the state importer, so
	// we need to figure the CSP using the import ID
	csp := getCspFromImportID(d.Id())
	if csp == "" {
		return errors.New("failed to get csp from import id")
	}

	orgCommitments, err := getOrganizationCommitments(ctx, meta)
	if err != nil {
		return err
	}
	assignments, err := getOrganizationCommitmentAssignments(ctx, meta)
	if err != nil {
		return err
	}
	assignmentsByCommitmentID := lo.GroupBy(assignments, func(a sdk.CastaiInventoryV1beta1CommitmentAssignment) string {
		return lo.FromPtr(a.CommitmentId)
	})

	var (
		gcpResources   []*GCPCUDResource
		azureResources []*AzureReservationResource
	)
	for _, c := range orgCommitments {
		c := c
		as := assignmentsByCommitmentID[lo.FromPtr(c.Id)]
		switch {
		case c.GcpResourceCudContext != nil:
			resource, err := MapCommitmentToCUDResource(c, as)
			if err != nil {
				return err
			}
			gcpResources = append(gcpResources, resource)
		case c.AzureReservationContext != nil:
			resource, err := MapCommitmentToReservationResource(c, as)
			if err != nil {
				return err
			}
			azureResources = append(azureResources, resource)
		}
	}

	switch csp {
	case "azure":
		reservations, reservationsOk, err := getReservationImportResources(d)
		if err != nil {
			return err
		}
		if reservationsOk {
			SortResources(azureResources, reservations)
		}
		if err := d.Set(FieldCommitmentsAzureReservations, azureResources); err != nil {
			return fmt.Errorf("setting azure reservations: %w", err)
		}
	case "gcp":
		cuds, cudsOk, err := getCUDImportResources(d)
		if err != nil {
			return err
		}
		if cudsOk {
			SortResources(gcpResources, cuds)
		}
		if err := d.Set(FieldCommitmentsGCPCUDs, gcpResources); err != nil {
			return fmt.Errorf("setting gcp cuds: %w", err)
		}
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

func getOrganizationCommitmentAssignments(
	ctx context.Context,
	meta any,
) ([]sdk.CastaiInventoryV1beta1CommitmentAssignment, error) {
	client := meta.(*ProviderConfig).api
	response, err := client.CommitmentsAPIGetCommitmentsAssignmentsWithResponse(ctx)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return nil, fmt.Errorf("fetching commitments: %w", checkErr)
	}
	if response.JSON200.CommitmentsAssignments == nil {
		return nil, nil
	}
	return *response.JSON200.CommitmentsAssignments, nil
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

func getCspFromImportID(id string) string {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
