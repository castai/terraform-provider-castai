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
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/commitments"
	"github.com/castai/terraform-provider-castai/castai/sdk"
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
			// Allow either reservations or CUDs - validated in the custom diff function
			commitments.FieldAzureReservationsCSV: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "CSV file containing reservations exported from Azure.",
			},
			commitments.FieldGCPCUDsJSON: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON file containing CUDs exported from GCP.",
			},
			commitments.FieldGCPCUDs: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of GCP CUDs.",
				Elem: &schema.Resource{
					Schema: commitments.GCPCUDResourceSchema,
				},
			},
			commitments.FieldGCPCUDConfigs: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of GCP CUD configurations.",
				Elem: &schema.Resource{
					Schema: commitments.GCPCUDConfigsSchema,
				},
			},
			commitments.FieldAzureReservations: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of Azure reservations.",
				Elem: &schema.Resource{
					Schema: commitments.AzureReservationResourceSchema,
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
		return fmt.Errorf("azure reservations are currently not supported")
	case cudsOk:
		return diff.SetNew(commitments.FieldGCPCUDs, cudResources)
	}

	return errors.New("unhandled combination of commitments input")
}

func getCUDImports(tfData resourceProvider) ([]sdk.CastaiInventoryV1beta1GCPCommitmentImport, bool, error) {
	cudsIface, ok := tfData.GetOk(commitments.FieldGCPCUDsJSON)
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

func getCUDConfigs(tfData resourceProvider) ([]*commitments.GCPCUDConfigResource, error) {
	var configs []*commitments.GCPCUDConfigResource
	if configsIface, ok := tfData.GetOk(commitments.FieldGCPCUDConfigs); ok {
		if err := mapstructure.Decode(configsIface, &configs); err != nil {
			return nil, err
		}
	}
	return configs, nil
}

// getCUDImportResources returns a slice of GCP CUD resources obtained from the input JSON.
func getCUDImportResources(tfData resourceProvider) ([]*commitments.GCPCUDResource, bool, error) {
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
		if err := c.Matcher.Validate(); err != nil {
			return nil, true, fmt.Errorf("invalid CUD matcher: %w", err)
		}
	}

	// Finally map the CUD imports to resources and combine them with the configurations
	res, err := commitments.MapConfiguredCUDImportsToResources(cuds, configs)
	if err != nil {
		return nil, true, err
	}
	return res, true, nil
}

// getCUDResources returns a slice of GCP CUD resources obtained from the state obtained from the API.
func getCUDResources(tfData resourceProvider) ([]*commitments.GCPCUDResource, bool, error) {
	cudsIface, ok := tfData.GetOk(commitments.FieldGCPCUDs)
	if !ok {
		return nil, false, nil
	}
	var res []*commitments.GCPCUDResource
	if err := mapstructure.Decode(cudsIface, &res); err != nil {
		return nil, true, err
	}
	return res, true, nil
}

func getReservationResources(tfData resourceProvider) ([]*commitments.AzureReservationResource, bool, error) {
	// TEMPORARY: support for Azure reservations will be added in one of the upcoming PRs
	_, ok := tfData.GetOk(commitments.FieldAzureReservationsCSV)
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
	organizationId, err := getOrganizationId(ctx, data, meta)
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

	data.SetId(organizationId)
	return nil
}

func deleteCommitments[R commitments.Resource](ctx context.Context, meta any, resources []R) error {
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
	organizationId, err := getOrganizationId(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, reservationsOk := data.GetOk(commitments.FieldAzureReservationsCSV)
	cuds, cudsOk, err := getCUDImports(data)

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

		cudsWithConfigs, err := commitments.MapConfigsToCUDs(
			lo.Map(gcpCommitments, func(item sdk.CastaiInventoryV1beta1Commitment, _ int) commitments.CastaiCommitment {
				return commitments.CastaiCommitment{CastaiInventoryV1beta1Commitment: item}
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
				commitments.MapCUDImportWithConfigToUpdateRequest(c),
			)
			if err := sdk.CheckOKResponse(res, err); err != nil {
				return diag.Errorf("updating commitment: %w", err)
			}
		}
	}

	data.SetId(organizationId)
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

	_, reservationsOk := d.GetOk(commitments.FieldAzureReservationsCSV)
	cuds, cudsOk, err := getCUDImportResources(d)
	if err != nil {
		return err
	}

	switch {
	case reservationsOk:
		return fmt.Errorf("azure reservations are currently not supported")
	case cudsOk:
		var resources []*commitments.GCPCUDResource
		for _, c := range orgCommitments {
			if c.GcpResourceCudContext == nil {
				continue
			}

			resource, err := commitments.MapCommitmentToCUDResource(c)
			if err != nil {
				return err
			}
			resources = append(resources, resource)
		}
		commitments.SortResources(resources, cuds)
		if err := d.Set(commitments.FieldGCPCUDs, resources); err != nil {
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
