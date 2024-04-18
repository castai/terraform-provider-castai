package castai

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				Description: "CSV file containing Azure reservations",
			},
			commitments.FieldGCPCUDsJSON: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON file containing GCP CUDs",
			},
			commitments.FieldGCPCUDs: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: commitments.GCPCUDResourceSchema,
				},
			},
			commitments.FieldAzureReservations: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: commitments.AzureReservationResourceSchema,
				},
			},
		},
	}
}

func commitmentsDiff(_ context.Context, diff *schema.ResourceDiff, _ any) error {
	reservationsCSV, reservationsOk := diff.GetOk(commitments.FieldAzureReservationsCSV)
	cudsJSON, cudsOk := diff.GetOk(commitments.FieldGCPCUDsJSON)
	if !reservationsOk && !cudsOk {
		return fmt.Errorf("one of 'azure_reservations_csv' or 'gcp_cuds_json' must be set")
	}
	if reservationsOk && cudsOk {
		return fmt.Errorf("either 'azure_reservations_csv' or 'gcp_cuds_json' can be set, not both")
	}

	switch {
	case reservationsOk:
		reservationResources, err := mapReservationsCSVToResources(reservationsCSV.(string))
		if err != nil {
			return err
		}
		return diff.SetNew(commitments.FieldAzureReservations, reservationResources)
	case cudsOk:
		cudResources, err := mapCUDsJSONToResources(cudsJSON.(string))
		if err != nil {
			return err
		}
		return diff.SetNew(commitments.FieldGCPCUDs, cudResources)
	}

	return errors.New("unhandled combination of commitments input")
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

	reservationsIface, reservationsOk := data.GetOk(commitments.FieldAzureReservations)
	cudsIface, cudsOk := data.GetOk(commitments.FieldGCPCUDs)

	switch {
	case reservationsOk:
		var reservations []*commitments.AzureReservationResource
		if err := mapstructure.Decode(reservationsIface, &reservations); err != nil {
			return diag.FromErr(err)
		}
		for _, c := range reservations {
			if c.Id == nil {
				return diag.Errorf("missing ID for Azure reservation")
			}
			if err := deleteCommitment(ctx, meta, *c.Id); err != nil {
				return diag.FromErr(err)
			}
		}
	case cudsOk:
		var cuds []*commitments.GCPCUDResource
		if err := mapstructure.Decode(cudsIface, &cuds); err != nil {
			return diag.FromErr(err)
		}
		for _, c := range cuds {
			if c.ID == nil {
				return diag.Errorf("missing ID for GCP CUD")
			}
			if err := deleteCommitment(ctx, meta, *c.ID); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	data.SetId(organizationId)
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

	reservationsCsv, reservationsOk := data.GetOk(commitments.FieldAzureReservationsCSV)
	cudsJSON, cudsOk := data.GetOk(commitments.FieldGCPCUDsJSON)

	switch {
	case reservationsOk:
		rows, err := parseCSV(reservationsCsv.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		imports, err := commitments.MapReservationsCSVRecordsToImports(rows)
		if err != nil {
			return diag.FromErr(err)
		}

		if err := importReservations(ctx, meta, imports); err != nil {
			return diag.FromErr(err)
		}
	case cudsOk:
		cuds, err := unmarshalCUDs(cudsJSON.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		if err := importCUDs(ctx, meta, cuds); err != nil {
			return diag.FromErr(err)
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

func importReservations(ctx context.Context, meta any, imports []sdk.CastaiInventoryV1beta1AzureReservationImport) error {
	res, err := meta.(*ProviderConfig).api.CommitmentsAPIImportAzureReservationsWithResponse(
		ctx,
		&sdk.CommitmentsAPIImportAzureReservationsParams{
			Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportAzureReservationsParamsBehaviour]("OVERWRITE"),
		},
		imports,
	)
	if checkErr := sdk.CheckOKResponse(res, err); checkErr != nil {
		return fmt.Errorf("upserting commitments: %w", checkErr)
	}
	return nil
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
	_, cudsOk := d.GetOk(commitments.FieldGCPCUDsJSON)

	switch {
	case reservationsOk:
		if err := d.Set(
			commitments.FieldAzureReservations,
			lo.Filter(orgCommitments, func(item sdk.CastaiInventoryV1beta1Commitment, index int) bool {
				return item.AzureReservationContext != nil
			}),
		); err != nil {
			return fmt.Errorf("setting azure reservations: %w", err)
		}
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
		if err := d.Set(commitments.FieldGCPCUDs, resources); err != nil {
			return fmt.Errorf("setting gcp cuds: %w", err)
		}
	}
	return nil
}

func parseCSV(val string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(val))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing commitments csv: %w", err)
	}
	return records, nil
}

func mapReservationsCSVToResources(csvStr string) ([]*commitments.AzureReservationResource, error) {
	records, err := parseCSV(csvStr)
	if err != nil {
		return nil, err
	}
	result, err := commitments.MapReservationCSVRecordsToResources(records)
	if err != nil {
		return nil, fmt.Errorf("parsing commitments csv: %w", err)
	}
	return result, nil
}

func mapCUDsJSONToResources(input string) ([]*commitments.GCPCUDResource, error) {
	cuds, err := unmarshalCUDs(input)
	if err != nil {
		return nil, err
	}

	res := make([]*commitments.GCPCUDResource, 0, len(cuds))
	for _, item := range cuds {
		v, err := commitments.MapCUDImportToResource(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
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
