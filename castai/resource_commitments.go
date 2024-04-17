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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			commitments.FieldReservationsOrganizationId: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the organization",
			},
			commitments.FieldGCPCUDs: {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{},
				},
			},
			commitments.FieldAzureReservations: {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						commitments.FieldReservationCount: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "amount of reserved instances",
						},
						commitments.FieldReservationInstanceType: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reserved instance type",
						},
						commitments.FieldReservationName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "unique reservation name in region for specific instance type",
						},
						commitments.FieldReservationPrice: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation price",
						},
						commitments.FieldReservationProvider: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation cloud provider (gcp, aws, azure)",
						},
						commitments.FieldReservationRegion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation region",
						},
						commitments.FieldReservationZoneId: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "reservation zone id",
						},
						commitments.FieldReservationZoneName: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "reservation zone name",
						},
						commitments.FieldReservationStartDate: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsRFC3339Time),
							Required:         true,
							Description:      "start date of reservation",
						},
						commitments.FieldReservationEndDate: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsRFC3339Time),
							Optional:         true,
							Description:      "end date of reservation",
						},
					},
				},
			},
		},
	}
}

func commitmentsDiff(_ context.Context, diff *schema.ResourceDiff, _ any) error {
	reservationsCSV, reservationsOk := diff.GetOk(commitments.FieldAzureReservationsCSV)
	cudsJSON, cudsOk := diff.GetOk(commitments.FieldAzureReservationsCSV)
	if !reservationsOk && !cudsOk {
		return fmt.Errorf("one of 'azure_reservations_csv' or 'gcp_cuds_json' must be set")
	}
	if reservationsOk && cudsOk {
		return fmt.Errorf("either 'azure_reservations_csv' or 'gcp_cuds_json' can be set, not both")
	}

	switch {
	case reservationsOk:
		reservationResources, err := mapReservationsCsvToCommitmentResources(reservationsCSV.(string))
		if err != nil {
			return err
		}
		return diff.SetNew(commitments.FieldAzureReservations, commitments.MapToCommitmentResourcesWithCommonFieldsOnly(reservationResources))
	case cudsOk:
		cudResources, err := mapCUDsJSONToCommitmentResources(cudsJSON.(string))
		if err != nil {
			return err
		}
		return diff.SetNew(commitments.FieldGCPCUDs, commitments.MapToCommitmentResourcesWithCommonFieldsOnly(cudResources))
	}

	return errors.New("unhandled combination of commitments input")
}

func commitmentsStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	organizationId, err := uuid.Parse(d.Id())
	if err != nil {
		return nil, err
	}

	if err := d.Set(commitments.FieldReservationsOrganizationId, organizationId.String()); err != nil {
		return nil, err
	}

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

func resourceCastaiCommitmentsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationId, err := getOrganizationId(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, reservationsOk := data.GetOk(commitments.FieldAzureReservations)
	_, cudsOk := data.GetOk(commitments.FieldGCPCUDs)

	switch {
	case reservationsOk:
		if err := upsertAzureReservations(ctx, meta, []sdk.CastaiInventoryV1beta1AzureReservationImport{}); err != nil {
			return diag.FromErr(err)
		}
	case cudsOk:
		if err := upsertGCPCUDs(ctx, meta, []sdk.CastaiInventoryV1beta1GCPCommitmentImport{}); err != nil {
			return diag.FromErr(err)
		}
	}

	data.SetId(organizationId)
	return nil
}

func resourceCastaiCommitmentsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Update commitments call start")
	defer log.Printf("[INFO] Update commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Create commitments call start")
	defer log.Printf("[INFO] Create commitments call end")

	return resourceCastaiCommitmentsUpsert(ctx, data, meta)
}

func resourceCastaiCommitmentsUpsert(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationId, err := getOrganizationId(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	reservationsCsv, reservationsOk := data.GetOk(commitments.FieldAzureReservationsCSV)
	cudsJSON, cudsOk := data.GetOk(commitments.FieldGCPCUDsJSON)

	switch {
	case reservationsOk:
		reservationResources, err := mapReservationsCsvToCommitmentResources(reservationsCsv.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		mappedReservations := lo.Map(reservationResources, func(item *commitments.CommitmentsResource, _ int) sdk.CastaiInventoryV1beta1AzureReservationImport {
			return commitments.MapCommitmentResourceToAzureReservationImport(*item)
		})

		if err := upsertAzureReservations(ctx, meta, mappedReservations); err != nil {
			return diag.FromErr(err)
		}
	case cudsOk:
		cuds, err := unmarshalCUDs(cudsJSON.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		if err := upsertGCPCUDs(ctx, meta, cuds); err != nil {
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

func upsertAzureReservations(ctx context.Context, meta any, cuds []sdk.CastaiInventoryV1beta1AzureReservationImport) error {
	res, err := meta.(*ProviderConfig).api.CommitmentsAPIImportAzureReservationsWithResponse(
		ctx,
		&sdk.CommitmentsAPIImportAzureReservationsParams{
			Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportAzureReservationsParamsBehaviour]("OVERWRITE"),
		},
		cuds,
	)
	if checkErr := sdk.CheckOKResponse(res, err); checkErr != nil {
		return fmt.Errorf("upserting commitments: %w", checkErr)
	}
	return nil
}

func upsertGCPCUDs(ctx context.Context, meta any, cuds []sdk.CastaiInventoryV1beta1GCPCommitmentImport) error {
	res, err := meta.(*ProviderConfig).api.CommitmentsAPIImportGCPCommitmentsWithResponse(
		ctx,
		&sdk.CommitmentsAPIImportGCPCommitmentsParams{
			Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportGCPCommitmentsParamsBehaviour]("OVERWRITE"),
		},
		cuds,
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
		if err := d.Set(
			commitments.FieldGCPCUDs,
			lo.Filter(orgCommitments, func(item sdk.CastaiInventoryV1beta1Commitment, index int) bool {
				return item.GcpResourceCudContext != nil
			}),
		); err != nil {
			return fmt.Errorf("setting gcp cuds: %w", err)
		}
	}
	return nil
}

func mapReservationsCsvToCommitmentResources(csvStr string) ([]*commitments.CommitmentsResource, error) {
	csvReader := csv.NewReader(strings.NewReader(csvStr))
	csvRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing commitments csv: %w", err)
	}

	result, err := commitments.MapCsvRecordsToReservationResources(csvRecords)
	if err != nil {
		return nil, fmt.Errorf("parsing commitments csv: %w", err)
	}

	return result, nil
}

func mapCUDsJSONToCommitmentResources(input string) ([]*commitments.CommitmentsResource, error) {
	return nil, nil
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

func getOrganizationCommitmentResources(ctx context.Context, meta any) ([]*commitments.CommitmentsResource, error) {
	cmts, err := getOrganizationCommitments(ctx, meta)
	if err != nil {
		return nil, err
	}
	return lo.Map(cmts, func(item sdk.CastaiInventoryV1beta1Commitment, _ int) *commitments.CommitmentsResource {
		return commitments.MapCommitmentToCommitmentsResource(item)
	}), nil
}
