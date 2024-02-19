package castai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldGKEClusterName          = "name"
	FieldGKEClusterProjectId     = "project_id"
	FieldGKEClusterLocation      = "location"
	FieldGKEClusterCredentialsId = "credentials_id"
	FieldGKEClusterCredentials   = "credentials_json"
)

func resourceGKECluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiGKEClusterCreate,
		ReadContext:   resourceCastaiGKEClusterRead,
		UpdateContext: resourceCastaiGKEClusterUpdate,
		DeleteContext: resourceCastaiClusterDelete,
		CustomizeDiff: clusterTokenDiff,
		Description:   "GKE cluster resource allows connecting an existing GKE cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(6 * time.Minute), // Cluster action timeout is 5 minutes.
		},

		Schema: map[string]*schema.Schema{
			FieldGKEClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GKE cluster name",
			},
			FieldGKEClusterCredentialsId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CAST AI credentials id for cluster",
			},
			FieldGKEClusterProjectId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP project id",
			},
			FieldGKEClusterLocation: {
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
			FieldGKEClusterCredentials: {
				Type:             schema.TypeString,
				Sensitive:        true,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP credentials.json from ServiceAccount with credentials for CAST AI",
			},
			FieldDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST.AI on disconnect",
			},
		},
	}
}

func resourceCastaiGKEClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if err := updateGKEClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Cluster with id %q has been registered, don't forget to install castai-agent helm chart", data.Id())

	return resourceCastaiGKEClusterRead(ctx, data, meta)
}

func resourceCastaiGKEClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if err := data.Set(FieldGKEClusterCredentialsId, toString(resp.JSON200.CredentialsId)); err != nil {
		return diag.FromErr(fmt.Errorf("setting credentials id: %w", err))
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
	}

	return nil
}

func resourceCastaiGKEClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateGKEClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiGKEClusterRead(ctx, data, meta)
}

func updateGKEClusterSettings(ctx context.Context, data *schema.ResourceData, client *sdk.ClientWithResponses) error {
	if !data.HasChanges(
		FieldGKEClusterCredentials,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{}

	credentialsJSON, ok := data.GetOk(FieldGKEClusterCredentials)
	if ok {
		req.Credentials = toPtr(credentialsJSON.(string))
	}

	if err := backoff.Retry(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
		if err != nil {
			return err
		}
		err = sdk.StatusOk(response)
		// In case of malformed user request return error to user right away.
		if response.StatusCode() == 400 && !sdk.IsCredentialsError(response) {
			return backoff.Permanent(err)
		}
		return err
	}, backoff.NewExponentialBackOff()); err != nil {
		return fmt.Errorf("updating cluster configuration: %w", err)
	}

	return nil
}
