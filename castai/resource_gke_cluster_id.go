package castai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldGKEClusterIdName      = "name"
	FieldGKEClusterIdProjectId = "project_id"
	FieldGKEClusterIdLocation  = "location"
	FieldGKEClientSA           = "client_service_account"
	FieldGKECastSA             = "cast_service_account"
)

func resourceGKEClusterId() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiGKEClusterIdCreate,
		ReadContext:   resourceCastaiGKEClusterIdRead,
		UpdateContext: resourceCastaiGKEClusterIdUpdate,
		DeleteContext: resourceCastaiGKEClusterIdDelete,
		CustomizeDiff: clusterTokenDiff,
		Description:   "GKE cluster resource allows connecting an existing GKE cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(6 * time.Minute), // Cluster action timeout is 5 minutes.
		},

		Schema: map[string]*schema.Schema{
			FieldGKEClusterIdName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GKE cluster name",
			},
			FieldGKEClusterIdProjectId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP project id",
			},
			FieldGKEClusterIdLocation: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP cluster zone in case of zonal or region in case of regional cluster",
			},
			FieldClusterToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "CAST.AI agent cluster token",
			},
			FieldGKEClientSA: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Service account email in client project",
			},
			FieldGKECastSA: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Service account email in cast project",
			},
		},
	}
}

func resourceCastaiGKEClusterIdCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldGKEClusterName).(string),
	}

	location := data.Get(FieldGKEClusterLocation).(string)
	region := location
	// Check if location is zone or location.
	if strings.Count(location, "-") > 1 {
		// region "europe-central2"
		// zone "europe-central2-a"
		regionParts := strings.Split(location, "-")
		regionParts = regionParts[:2]
		region = strings.Join(regionParts, "-")
	}

	req.Gke = &sdk.ExternalclusterV1GKEClusterParams{
		ProjectId:   toPtr(data.Get(FieldGKEClusterProjectId).(string)),
		Region:      &region,
		Location:    &location,
		ClusterName: toPtr(data.Get(FieldGKEClusterName).(string)),
	}

	log.Printf("[INFO] Registering new external cluster: %#v", req)
	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	tkn, err := createClusterToken(ctx, client, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := data.Set(FieldClusterToken, tkn); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster token: %w", err))
	}
	data.SetId(clusterID)
	// If client service account is set, create service account on cast side.
	if len(data.Get(FieldGKEClientSA).(string)) > 0 {
		resp, err := client.ExternalClusterAPIGKECreateSAWithResponse(ctx, data.Id(), sdk.ExternalClusterAPIGKECreateSARequest{
			Gke: &sdk.ExternalclusterV1UpdateGKEClusterParams{
				GkeSaImpersonate: toPtr(data.Get(FieldGKEClientSA).(string)),
				ProjectId:        toPtr(data.Get(FieldGKEClusterProjectId).(string)),
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
		if resp.JSON200 == nil || resp.JSON200.ServiceAccount == nil {
			return diag.FromErr(fmt.Errorf("service account not returned"))
		}
		if err := data.Set(FieldGKECastSA, toString(resp.JSON200.ServiceAccount)); err != nil {
			return diag.FromErr(fmt.Errorf("service account id: %w", err))
		}
	}
	return nil
}

func resourceCastaiGKEClusterIdRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		log.Printf("[INFO] id is null not fetching anything.")
		return nil
	}

	log.Printf("[INFO] Getting cluster information.")
	resp, err := fetchClusterData(ctx, client, data.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		data.SetId("")
		return nil
	}
	if GKE := resp.JSON200.Gke; GKE != nil {
		if err := data.Set(FieldGKEClusterProjectId, toString(GKE.ProjectId)); err != nil {
			return diag.FromErr(fmt.Errorf("setting project id: %w", err))
		}
		if err := data.Set(FieldGKEClusterLocation, toString(GKE.Location)); err != nil {
			return diag.FromErr(fmt.Errorf("setting location: %w", err))
		}
		if err := data.Set(FieldGKEClusterName, toString(GKE.ClusterName)); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster name: %w", err))
		}
		if err := data.Set(FieldGKEClientSA, toString(GKE.ClientServiceAccount)); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster client sa email: %w", err))
		}
	}
	return nil
}

func resourceCastaiGKEClusterIdUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceCastaiGKEClusterIdRead(ctx, data, meta)
}

func resourceCastaiGKEClusterIdDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceCastaiClusterDelete(ctx, data, meta)
}
