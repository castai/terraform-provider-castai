package castai

import (
	"bytes"
	"context"
	"encoding/json"
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
	FieldGKEClusterName        = "name"
	FieldGKEClusterProjectId   = "project_id"
	FieldGKEClusterLocation    = "location"
	FieldGKEClusterCredentials = "credentials_json"

	FieldGKECredentials                          = "credentials"
	FieldGKECredentialsTargetServiceAccountEmail = "target_service_account_email"
	FieldGKECredentialsCastaiServiceAccountEmail = "castai_service_account_email"
)

type gkeCredentialsMode int

const (
	gkeCredentialsModeLegacy gkeCredentialsMode = iota + 1
	gkeCredentialsModeImpersonation
	gkeCredentialsModeNone
)

func resourceGKECluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiGKEClusterCreate,
		ReadContext:   resourceCastaiGKEClusterRead,
		UpdateContext: resourceCastaiGKEClusterUpdate,
		DeleteContext: resourceCastaiGKEClusterDelete,
		Description:   "GKE cluster resource allows connecting an existing GKE cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldGKEClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GKE cluster name",
			},
			FieldClusterCredentialsId: {
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
				Deprecated:       "use the credentials block instead",
				ConflictsWith:    []string{FieldGKECredentials},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP credentials.json from ServiceAccount with credentials for CAST AI",
			},
			FieldGKECredentials: {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{FieldGKEClusterCredentials},
				Description:   "GKE cluster credentials. Configure GKE service account impersonation.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldGKECredentialsTargetServiceAccountEmail: {
							Type:             schema.TypeString,
							Optional:         true,
							RequiredWith:     []string{FieldGKECredentials + ".0." + FieldGKECredentialsCastaiServiceAccountEmail},
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							Description:      "Service account email in the GCP project that CAST AI impersonates",
						},
						FieldGKECredentialsCastaiServiceAccountEmail: {
							Type:             schema.TypeString,
							Optional:         true,
							RequiredWith:     []string{FieldGKECredentials + ".0." + FieldGKECredentialsTargetServiceAccountEmail},
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							Description:      "CAST AI service account email used to access the target project",
						},
					},
				},
			},
			FieldDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST.AI on disconnect",
			},
			FieldClusterOrganizationId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CAST AI organization ID",
			},
		},
	}
}

// gkeCredentialsModeFromData reports which credentials configuration the resource uses.
func gkeCredentialsModeFromData(data *schema.ResourceData) gkeCredentialsMode {
	if _, ok := data.GetOk(FieldGKEClusterCredentials); ok {
		return gkeCredentialsModeLegacy
	}
	block := gkeCredentialsBlock(data)
	if block == nil {
		return gkeCredentialsModeNone
	}
	// Values are read from either config or shimmed state; both may materialize
	// all block fields with "" for unset ones, so match on non-empty values.
	for _, field := range []string{FieldGKECredentialsTargetServiceAccountEmail, FieldGKECredentialsCastaiServiceAccountEmail} {
		if v, _ := block[field].(string); v != "" {
			return gkeCredentialsModeImpersonation
		}
	}
	return gkeCredentialsModeNone
}

func gkeCredentialsBlock(data *schema.ResourceData) map[string]interface{} {
	raw, ok := data.GetOk(FieldGKECredentials)
	if !ok {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	block, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return block
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

	if gkeCredentialsModeFromData(data) == gkeCredentialsModeImpersonation {
		if err := applyGKECredentialsImpersonation(ctx, client, data); err != nil {
			return diag.FromErr(err)
		}
		if err := refreshGKEClusterCredentialsID(ctx, client, data); err != nil {
			return diag.FromErr(err)
		}
	} else if err := updateGKEClusterSettings(ctx, data, client); err != nil {
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

	if resp.JSON200.CredentialsId != nil && *resp.JSON200.CredentialsId != data.Get(FieldClusterCredentialsId) {
		switch gkeCredentialsModeFromData(data) {
		case gkeCredentialsModeLegacy:
			log.Printf("[WARN] Drift in credentials from state (%q) and in API (%q), resetting credentials JSON to force re-applying credentials from configuration",
				data.Get(FieldClusterCredentialsId), *resp.JSON200.CredentialsId)
			if err := data.Set(FieldGKEClusterCredentials, "credentials-drift-detected-force-apply"); err != nil {
				return diag.FromErr(fmt.Errorf("setting client ID: %w", err))
			}
		case gkeCredentialsModeImpersonation:
			// No fake needed: impersonation drift is surfaced by comparing the
			// configured target_service_account with the API clientServiceAccount below.
			log.Printf("[WARN] Drift in credentials from state (%q) and in API (%q), comparing impersonated service account",
				data.Get(FieldClusterCredentialsId), *resp.JSON200.CredentialsId)
		}
	}

	if err := data.Set(FieldClusterCredentialsId, toString(resp.JSON200.CredentialsId)); err != nil {
		return diag.FromErr(fmt.Errorf("setting credentials id: %w", err))
	}
	if err := data.Set(FieldClusterOrganizationId, toString(resp.JSON200.OrganizationId)); err != nil {
		return diag.FromErr(fmt.Errorf("setting organization id: %w", err))
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
		if gkeCredentialsModeFromData(data) == gkeCredentialsModeImpersonation && GKE.ClientServiceAccount != nil {
			apiTarget := toString(GKE.ClientServiceAccount)
			cfgTarget, _ := gkeCredentialsBlock(data)[FieldGKECredentialsTargetServiceAccountEmail].(string)
			if apiTarget != cfgTarget {
				log.Printf("[WARN] Drift in impersonated service account from state (%q) and in API (%q), resetting target_service_account to force re-applying credentials from configuration", cfgTarget, apiTarget)
				if err := setGKECredentialsBlockField(data, FieldGKECredentialsTargetServiceAccountEmail, apiTarget); err != nil {
					return diag.FromErr(fmt.Errorf("setting target service account: %w", err))
				}
			}
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

func updateGKEClusterSettings(ctx context.Context, data *schema.ResourceData, client sdk.ClientWithResponsesInterface) error {
	if !data.HasChanges(
		FieldGKEClusterCredentials,
		FieldGKECredentials,
		FieldClusterCredentialsId,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	if gkeCredentialsModeFromData(data) == gkeCredentialsModeImpersonation {
		if err := applyGKECredentialsImpersonation(ctx, client, data); err != nil {
			return err
		}
		return refreshGKEClusterCredentialsID(ctx, client, data)
	}

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{}
	if gkeCredentialsModeFromData(data) == gkeCredentialsModeLegacy {
		if credentialsJSON, ok := data.GetOk(FieldGKEClusterCredentials); ok {
			req.Credentials = toPtr(credentialsJSON.(string))
		}
	}

	return resourceCastaiClusterUpdate(ctx, client, data, &req)
}

func applyGKECredentialsImpersonation(ctx context.Context, client sdk.ClientWithResponsesInterface, data *schema.ResourceData) error {
	block := gkeCredentialsBlock(data)
	body, err := json.Marshal(struct {
		Gke *struct {
			GkeSaImpersonate   *string `json:"gkeSaImpersonate,omitempty"`
			CastServiceAccount *string `json:"castServiceAccount,omitempty"`
		} `json:"gke,omitempty"`
	}{
		Gke: &struct {
			GkeSaImpersonate   *string `json:"gkeSaImpersonate,omitempty"`
			CastServiceAccount *string `json:"castServiceAccount,omitempty"`
		}{
			GkeSaImpersonate:   toPtr(block[FieldGKECredentialsTargetServiceAccountEmail].(string)),
			CastServiceAccount: toPtr(block[FieldGKECredentialsCastaiServiceAccountEmail].(string)),
		},
	})
	if err != nil {
		return fmt.Errorf("building GKE impersonation service account request: %w", err)
	}
	resp, err := client.ExternalClusterAPIGKECreateSAWithBodyWithResponse(ctx, data.Id(), "application/json", bytes.NewReader(body))
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return fmt.Errorf("creating GKE impersonation service account: %w", checkErr)
	}
	return nil
}

func refreshGKEClusterCredentialsID(ctx context.Context, client sdk.ClientWithResponsesInterface, data *schema.ResourceData) error {
	resp, err := fetchClusterData(ctx, client, data.Id())
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if resp.JSON200.CredentialsId != nil {
		if err := data.Set(FieldClusterCredentialsId, *resp.JSON200.CredentialsId); err != nil {
			return fmt.Errorf("setting credentials id: %w", err)
		}
	}
	return nil
}

func setGKECredentialsBlockField(data *schema.ResourceData, field, value string) error {
	block := gkeCredentialsBlock(data)
	if block == nil {
		return nil
	}
	updated := make(map[string]interface{}, len(block)+1)
	for k, v := range block {
		updated[k] = v
	}
	updated[field] = value
	return data.Set(FieldGKECredentials, []interface{}{updated})
}

func resourceCastaiGKEClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Disable service account used for impersonation.
	client := meta.(*ProviderConfig).api
	log.Printf("[INFO] Disabling service account.")
	_, err := client.ExternalClusterAPIDisableGKESA(ctx, data.Id())
	if err != nil {
		log.Printf("[ERROR] Failed to disable service account: %v", err)
	}
	return resourceCastaiClusterDelete(ctx, data, meta)
}
