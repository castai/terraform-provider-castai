package castai

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/castai/terraform-provider-castai/castai/reservations"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceReservations() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiReservationsRead,
		CreateContext: resourceCastaiReservationsCreate,
		UpdateContext: resourceCastaiReservationsUpdate,
		DeleteContext: resourceCastaiReservationsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: reservationsStateImporter,
		},
		Description:        "Reservation represents cloud service provider reserved instances that can be used by CAST AI autoscaler.",
		DeprecationMessage: "Use castai_commitments resource instead.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},
		CustomizeDiff: reservationsDiff,
		Schema: map[string]*schema.Schema{
			reservations.FieldReservationsCSV: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "csv file containing reservations",
			},
			reservations.FieldReservationsOrganizationId: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "organization",
			},
			reservations.FieldReservations: {
				Type:     schema.TypeList,
				Computed: true,
				Optional: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						reservations.FieldReservationCount: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "amount of reserved instances",
						},
						reservations.FieldReservationInstanceType: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reserved instance type",
						},
						reservations.FieldReservationName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "unique reservation name in region for specific instance type",
						},
						reservations.FieldReservationPrice: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation price",
						},
						reservations.FieldReservationProvider: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation cloud provider (gcp, aws, azure)",
						},
						reservations.FieldReservationRegion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "reservation region",
						},
						reservations.FieldReservationZoneId: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "reservation zone id",
						},
						reservations.FieldReservationZoneName: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "reservation zone name",
						},
						reservations.FieldReservationStartDate: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsRFC3339Time),
							Required:         true,
							Description:      "start date of reservation",
						},
						reservations.FieldReservationEndDate: {
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

func reservationsDiff(_ context.Context, diff *schema.ResourceDiff, _ any) error {
	reservationsCsv, _ := diff.GetOk(reservations.FieldReservationsCSV)
	reservationResources, err := mapReservationsCsvToReservationResources(reservationsCsv.(string))
	if err != nil {
		return err
	}

	return diff.SetNew(reservations.FieldReservations, reservations.MapToReservationResourcesWithCommonFieldsOnly(reservationResources))
}

func reservationsStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	organizationId, err := uuid.Parse(d.Id())
	if err != nil {
		return nil, err
	}

	if err := d.Set(reservations.FieldReservationsOrganizationId, organizationId.String()); err != nil {
		return nil, err
	}

	if err := populateReservationsResourceData(ctx, d, meta, organizationId.String()); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func resourceCastaiReservationsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Info(ctx, "Get reservations call start")
	defer tflog.Info(ctx, "Get reservations call end")

	organizationId, err := getOrganizationId(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := populateReservationsResourceData(ctx, d, meta, organizationId); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCastaiReservationsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationId, err := getOrganizationId(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = upsertReservations(ctx, meta, organizationId, []*reservations.ReservationResource{})
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(organizationId)
	return nil
}

func resourceCastaiReservationsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Update reservations call start")
	defer tflog.Info(ctx, "Update reservations call end")

	return resourceCastaiReservationsUpsert(ctx, data, meta)
}

func resourceCastaiReservationsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Create reservations call start")
	defer tflog.Info(ctx, "Create reservations call end")

	return resourceCastaiReservationsUpsert(ctx, data, meta)
}

func resourceCastaiReservationsUpsert(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationId, err := getOrganizationId(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	reservationsCsv, _ := data.GetOk(reservations.FieldReservationsCSV)
	reservationResources, err := mapReservationsCsvToReservationResources(reservationsCsv.(string))
	if err != nil {
		return diag.FromErr(err)
	}

	err = upsertReservations(ctx, meta, organizationId, reservationResources)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(organizationId)

	return resourceCastaiReservationsRead(ctx, data, meta)
}

func upsertReservations(ctx context.Context, meta interface{}, organizationId string, reservationResources []*reservations.ReservationResource) error {
	client := meta.(*ProviderConfig).api
	mappedReservations := lo.Map(reservationResources, func(item *reservations.ReservationResource, _ int) sdk.CastaiInventoryV1beta1GenericReservation {
		return reservations.MapReservationResourceToGenericReservation(*item)
	})

	response, err := client.InventoryAPIOverwriteReservationsWithResponse(ctx, organizationId, sdk.InventoryAPIOverwriteReservationsJSONRequestBody{
		Items: &mappedReservations,
	})
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return fmt.Errorf("upserting reservations: %w", checkErr)
	}

	return nil
}

func populateReservationsResourceData(ctx context.Context, d *schema.ResourceData, meta any, organizationId string) error {
	organizationReservations, err := getOrganizationReservationResources(ctx, meta, organizationId)
	if err != nil {
		return err
	}

	if err := d.Set(reservations.FieldReservations, organizationReservations); err != nil {
		return fmt.Errorf("setting reservations: %w", err)
	}

	return nil
}

func mapReservationsCsvToReservationResources(reservationsCsv string) ([]*reservations.ReservationResource, error) {
	csvReader := csv.NewReader(strings.NewReader(reservationsCsv))
	csvRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing reservations csv: %w", err)
	}

	result, err := reservations.MapCsvRecordsToReservationResources(csvRecords)
	if err != nil {
		return nil, fmt.Errorf("parsing reservations csv: %w", err)
	}

	return result, nil
}

func getOrganizationReservationResources(ctx context.Context, meta any, organizationId string) ([]*reservations.ReservationResource, error) {
	client := meta.(*ProviderConfig).api

	response, err := client.InventoryAPIGetReservationsWithResponse(ctx, organizationId)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return nil, fmt.Errorf("fetching reservations: %w", checkErr)
	}

	return lo.Map(*response.JSON200.Reservations, func(item sdk.CastaiInventoryV1beta1ReservationDetails, _ int) *reservations.ReservationResource {
		return reservations.MapReservationDetailsToReservationResource(item)
	}), nil
}

func getOrganizationId(ctx context.Context, d *schema.ResourceData, meta any) (string, error) {
	client := meta.(*ProviderConfig).api

	organizationUid, found := d.GetOk(reservations.FieldReservationsOrganizationId)
	if found {
		organizationId, err := uuid.Parse(organizationUid.(string))
		if err != nil {
			return "", err
		}

		return organizationId.String(), nil
	}

	response, err := client.UsersAPIListOrganizationsWithResponse(ctx, &sdk.UsersAPIListOrganizationsParams{})
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return "", fmt.Errorf("fetching organizations: %w", checkErr)
	}

	if len(response.JSON200.Organizations) > 1 {
		return "", fmt.Errorf("found more than 1 organization, you can specify exact organization using 'organization_id' attribute")
	}

	for _, organization := range response.JSON200.Organizations {
		return *organization.Id, nil
	}

	return "", fmt.Errorf("no organizations found")
}
