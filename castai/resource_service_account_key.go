package castai

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldServiceAccountKeyID               = "id"
	FieldServiceAccountKeyOrganizationID   = "organization_id"
	FieldServiceAccountKeyServiceAccountID = "service_account_id"
	FieldServiceAccountKeyName             = "name"
	FieldServiceAccountKeyPrefix           = "prefix"
	FieldServiceAccountKeyLastUsedAt       = "last_used_at"
	FieldServiceAccountKeyExpiresAt        = "expires_at"
	FieldServiceAccountKeyActive           = "active"
	FieldServiceAccountKeyToken            = "token"
)

func resourceServiceAccountKey() *schema.Resource {
	return &schema.Resource{
		Description:   "Service account key resource allows managing CAST AI service account keys.",
		CreateContext: resourceServiceAccountKeyCreate,
		ReadContext:   resourceServiceAccountKeyRead,
		UpdateContext: resourceServiceAccountKeyUpdate,
		DeleteContext: resourceServiceAccountKeyDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldServiceAccountKeyOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "ID of the organization.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldServiceAccountKeyServiceAccountID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "ID of the service account.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldServiceAccountKeyName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Name of the service account key.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldServiceAccountKeyPrefix: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prefix of the service account key.",
			},
			FieldServiceAccountKeyLastUsedAt: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last time the service account key was used.",
			},
			FieldServiceAccountKeyExpiresAt: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "",
				ForceNew:         true,
				Description:      "The expiration time of the service account key in RFC3339 format. Defaults to an empty string.",
				ValidateDiagFunc: validateRFC3339TimeOrEmpty,
			},
			FieldServiceAccountKeyActive: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the service account key is active. Defaults to true.",
			},
			FieldServiceAccountKeyToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The token of the service account key used for authentication.",
			},
		},
	}
}

func resourceServiceAccountKeyRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		return diag.Errorf("service account key ID is not set")
	}

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	if serviceAccountID == "" {
		return diag.Errorf("service account ID is not set")
	}
	serviceAccountKeyID := data.Id()

	logKeys := map[string]interface{}{
		"resource_id":        serviceAccountKeyID,
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
	}

	tflog.Info(ctx, "reading service account key", logKeys)

	resp, err := client.ServiceAccountsAPIGetServiceAccountKeyWithResponse(ctx, organizationID, serviceAccountID, serviceAccountKeyID)
	if err != nil {
		return diag.Errorf("reading service account key: %v", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "resource is not found, removing from state", logKeys)
		data.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("reading service account key: %v", err)
	}

	tflog.Info(ctx, "found service account key", logKeys)

	serviceAccountKey := resp.JSON200

	if err := data.Set(FieldServiceAccountKeyOrganizationID, organizationID); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyOrganizationID, err)
	}

	if err := data.Set(FieldServiceAccountKeyServiceAccountID, serviceAccountID); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyServiceAccountID, err)
	}

	if err := data.Set(FieldServiceAccountKeyName, serviceAccountKey.Key.Name); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyName, err)
	}

	if err := data.Set(FieldServiceAccountKeyPrefix, serviceAccountKey.Key.Prefix); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyPrefix, err)
	}

	if serviceAccountKey.Key.LastUsedAt != nil {
		if err := data.Set(FieldServiceAccountKeyLastUsedAt, serviceAccountKey.Key.LastUsedAt.Format(time.RFC3339)); err != nil {
			return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyLastUsedAt, err)
		}
	}

	if serviceAccountKey.Key.ExpiresAt != nil {
		if err := data.Set(FieldServiceAccountKeyExpiresAt, serviceAccountKey.Key.ExpiresAt.Format(time.RFC3339)); err != nil {
			return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyExpiresAt, err)
		}
	}

	if err := data.Set(FieldServiceAccountKeyActive, serviceAccountKey.Key.Active); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyActive, err)
	}
	return nil
}

func resourceServiceAccountKeyCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var expiresAtTime *time.Time

	client := meta.(*ProviderConfig).api

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	name := data.Get(FieldServiceAccountKeyName).(string)
	expiresAt := data.Get(FieldServiceAccountKeyExpiresAt).(string)
	active := data.Get(FieldServiceAccountKeyActive).(bool)

	if expiresAt != "" {
		expiresAtParsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return diag.Errorf("parsing expires_at date: %v", err)
		}
		expiresAtTime = &expiresAtParsed
	}

	logKeys := map[string]interface{}{
		"name":               name,
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
	}

	tflog.Info(ctx, "creating service account key", logKeys)

	resp, err := client.ServiceAccountsAPICreateServiceAccountKeyWithResponse(
		ctx,
		organizationID,
		serviceAccountID,
		sdk.ServiceAccountsAPICreateServiceAccountKeyRequest{
			Key: sdk.CastaiServiceaccountsV1beta1CreateServiceAccountKeyRequestKey{
				Active:    &active,
				ExpiresAt: expiresAtTime,
				Name:      name,
			},
		},
	)
	if err := sdk.CheckResponseCreated(resp, err); err != nil {
		return diag.Errorf("creating service account key: %v", err)
	}

	if resp.JSON201 == nil {
		return diag.Errorf("creating service account key: response is missing")
	}

	if resp.JSON201.Id == nil {
		return diag.Errorf("creating service account key: id is missing")
	}

	if resp.JSON201.Token == nil {
		return diag.Errorf("creating service account key: token is missing")
	}

	serviceAccountKeyID := resp.JSON201.Id

	logKeys["resource_id"] = serviceAccountKeyID
	tflog.Info(ctx, "created service account key", logKeys)

	data.SetId(*serviceAccountKeyID)

	if err := data.Set(FieldServiceAccountKeyToken, *resp.JSON201.Token); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyToken, err)
	}
	return resourceServiceAccountKeyRead(ctx, data, meta)
}

func resourceServiceAccountKeyUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	keyID := data.Id()
	active := data.Get(FieldServiceAccountKeyActive).(bool)

	logKeys := map[string]interface{}{
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
		"resource_id":        keyID,
	}

	tflog.Info(ctx, "updating service account key", logKeys)

	resp, err := client.ServiceAccountsAPIUpdateServiceAccountKeyWithResponse(ctx, organizationID, serviceAccountID, keyID, sdk.KeyIsTheServiceAccountKeyToUpdate{
		Active: active,
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating service account key: %v", err)
	}

	tflog.Info(ctx, "updated service account key", logKeys)

	return resourceServiceAccountKeyRead(ctx, data, meta)
}

func resourceServiceAccountKeyDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	keyID := data.Id()

	logKeys := map[string]interface{}{
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
		"resource_id":        keyID,
	}

	tflog.Info(ctx, "deleting service account key", logKeys)

	resp, err := client.ServiceAccountsAPIDeleteServiceAccountKey(ctx, organizationID, serviceAccountID, keyID)
	if err := sdk.CheckRawResponseNoContent(resp, err); err != nil {
		return diag.Errorf("deleting service account key: %v", err)
	}

	tflog.Info(ctx, "deleted service account key", logKeys)

	data.SetId("")

	return nil
}
