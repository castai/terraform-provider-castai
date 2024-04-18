package castai

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						commitments.FieldId: {
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
						commitments.FieldStatus: {
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
						commitments.FieldStartTimestamp: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldEndTimestamp: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldRegion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldCPU: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "",
						},
						commitments.FieldMemoryMB: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "",
						},
						commitments.FieldPlan: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldType: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
					},
				},
			},
			commitments.FieldAzureReservations: {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						commitments.FieldExpirationDate: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldProductName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldPurchaseDate: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldQuantity: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldRegion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldReservationId: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldScope: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldScopeResourceGroup: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldScopeSubscription: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldStatus: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldTerm: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						commitments.FieldType: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
					},
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
	log.Printf("[INFO] Get commitments call start")
	defer log.Printf("[INFO] Get commitments call end")

	if err := populateCommitmentsResourceData(ctx, d, meta); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceCastaiCommitmentsDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	_, reservationsOk := data.GetOk(commitments.FieldAzureReservations)
	_, cudsOk := data.GetOk(commitments.FieldGCPCUDs)

	switch {
	case reservationsOk:
		if err := importReservations(ctx, meta, []sdk.CastaiInventoryV1beta1AzureReservationImport{}); err != nil {
			return diag.FromErr(err)
		}
	case cudsOk:
		if err := importCUDs(ctx, meta, []sdk.CastaiInventoryV1beta1GCPCommitmentImport{}); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceCastaiCommitmentsUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	log.Printf("[INFO] Update commitments call start")
	defer log.Printf("[INFO] Update commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	log.Printf("[INFO] Create commitments call start")
	defer log.Printf("[INFO] Create commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsUpsert(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
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
